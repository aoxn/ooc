package rolling

import (
	"context"
	"fmt"
	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

func (r *ReconcileRolling) setDefault(cxt context.Context, rolling *acv1.Rolling) {
	if rolling.Status.StartTime == nil {
		r.recorder.Event(rolling, v1.EventTypeNormal, "Start", "start to process rolling job")
		now := metav1.Now()
		rolling.Status.StartTime = &now
	}

	if rolling.Spec.MaxParallel == 0 {
		rolling.Spec.MaxParallel = 1
	}

	if rolling.Spec.Type == "" {
		rolling.Spec.Type = "task"
	}
	return
}

func (r *ReconcileRolling) manage(cxt context.Context, rolling *acv1.Rolling) error {

	r.setDefault(cxt, rolling)

	if rolling.Status.Phase == acv1.PhaseCompleted || rolling.Status.Phase == acv1.PhaseFailed {
		klog.Infof("Rolling is %s phase, skip reconciling", rolling.Status.Phase)
		return nil
	}

	if rolling.Spec.Paused {
		if rolling.Status.Phase != acv1.PhasePaused {
			r.recorder.Event(rolling, v1.EventTypeNormal, "Paused", fmt.Sprint("rolling job is paused"))
			rolling.Status.Phase = acv1.PhasePaused
		}
		//if rolling.Status.Active == 0 {
		//	return r.updateRollingStatus(cxt, rolling)
		//}
	} else {
		if rolling.Status.Phase == acv1.PhasePaused {
			rolling.Status.Phase = acv1.PhaseReconciling
			r.recorder.Event(rolling, v1.EventTypeNormal, "Running", fmt.Sprint("rolling job continue"))
		}
	}

	m := NewRollingTaskManager(r, cxt, rolling)
	if m == nil {
		klog.Errorf("Can't find task manager for rolling %s", rolling.Name)
		return nil
	}

	nodes, err := r.getNodes(cxt, rolling)
	if err != nil {
		klog.Error(err, "Failed to getNodes, %++v", err)
		return err
	}

	rolling.Status.Active, rolling.Status.Failed, rolling.Status.Succeeded, err = m.StatusStatistics(nodes)
	if err != nil {
		klog.Error(err, "Failed to nodesToTasks, %++v", err)
		return err
	}
	rolling.Status.Total = int32(len(nodes))

	if rolling.Status.Phase == acv1.PhasePaused {
		klog.Infof("Rolling is %s phase, skip reconciling", rolling.Status.Phase)
		return r.updateRollingStatus(cxt, rolling)
	}

	if rolling.Status.Active > 0 {
		klog.Infof("Rolling has %d active task, skip reconciling", rolling.Status.Active)
		return r.updateRollingStatus(cxt, rolling)
	}

	rolling.Status.Phase = acv1.PhaseReconciling

	if rolling.Status.Succeeded == rolling.Status.Total {
		klog.Infof("Rolling is completed")
		rolling.Status.Phase = acv1.PhaseCompleted
		now := metav1.Now()
		rolling.Status.CompletionTime = &now
		r.recorder.Event(rolling, v1.EventTypeNormal, "Completed", fmt.Sprint("rolling job is completed"))
		m.CleanupTasks()
		return r.updateRollingStatus(cxt, rolling)
	}

	if rolling.Status.Failed > 0 {
		if rolling.Status.Failed > rolling.Spec.MaxUnavailable || rolling.Status.Failed == rolling.Status.Total {
			r.recorder.Event(rolling, v1.EventTypeWarning, "Failed", fmt.Sprint("rolling job is failed"))
			rolling.Status.Phase = acv1.PhaseFailed
			return r.updateRollingStatus(cxt, rolling)
		}

		switch rolling.Spec.FailurePolicy {
		case acv1.FailurePolicyFailed, "":
			r.recorder.Event(rolling, v1.EventTypeWarning, "Failed", fmt.Sprint("rolling job is failed"))
			rolling.Status.Phase = acv1.PhaseFailed
			return r.updateRollingStatus(cxt, rolling)
		case acv1.FailurePolicyPause:
			r.recorder.Event(rolling, v1.EventTypeWarning, "Paused", fmt.Sprint("rolling job is pause because of failed task"))
			rolling.Status.Phase = acv1.PhasePaused
			rolling.Spec.Paused = true
			return r.updateRollingStatusAndPause(cxt, rolling)
		case acv1.FailurePolicyContinue:

		}
	}

	batchSize := r.batchSize(rolling)
	var wg sync.WaitGroup
	klog.Infof("rolling will start %d tasks", batchSize)
	r.recorder.Event(rolling, v1.EventTypeNormal, "Created", fmt.Sprintf("rolling job create %d tasks", batchSize))
	for _, node := range nodes {
		if batchSize <= 0 {
			break
		}
		if node != nil {
			if run := m.ShouldRunTask(node); run {
				batchSize--
				rolling.Status.Active++
				wg.Add(1)
				go func(nodeMeta *v1.Node) {
					klog.Infof("rolling create task on node (%s)", node.Name)
					err := m.RunTask(nodeMeta)
					if err != nil {
						klog.Error(err, "Failed to create task")
					}
					wg.Done()
				}(node)
			}
		}
	}
	wg.Wait()
	klog.Infof("rolling created %d tasks", batchSize)

	return r.updateRollingStatus(cxt, rolling)

}

func (r *ReconcileRolling) updateRollingStatusAndPause(cxt context.Context, rolling *acv1.Rolling) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"paused": %s}}`, rolling.Spec.Paused))
	if err := r.client.Patch(cxt, rolling, client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
		klog.Errorf("Failed to patch rolling instance %s, %++v", rolling.Name, err)
		return err
	}
	return r.updateRollingStatus(cxt, rolling)
}

func (r *ReconcileRolling) cleanupTasks(cxt context.Context, tasks map[string]acv1.Task) error {
	for i, _ := range tasks {
		task := tasks[i]
		for i := 0; i <= 3; i++ {
			if err := r.client.Delete(cxt, &task); err == nil {
				break
			}
		}
	}
	return nil
}

func (r *ReconcileRolling) batchSize(rolling *acv1.Rolling) int32 {
	batchSize := rolling.Spec.MaxParallel - rolling.Status.Active
	if rolling.Spec.SlowStart {
		batchSize = rolling.Status.CurrentMaxParallel * 2
	}

	if batchSize > rolling.Spec.MaxParallel {
		batchSize = rolling.Spec.MaxParallel
	}

	rolling.Status.CurrentMaxParallel = batchSize
	return batchSize
}

func (r *ReconcileRolling) updateRollingStatus(cxt context.Context, rolling *acv1.Rolling) error {
	var err error
	for i := 0; i <= 3; i = i + 1 {
		latestRollingJob := &acv1.Rolling{}

		err = r.client.Get(cxt, types.NamespacedName{Name: rolling.Name, Namespace: rolling.Namespace}, latestRollingJob)
		if err != nil {
			klog.Errorf("Failed to get rolling instance %s, %++v", rolling.Name, err)
			break
		}
		n := latestRollingJob.DeepCopy()
		n.Status = rolling.Status
		klog.Infof("latestRollingJob is %s, %s, status: %++v", latestRollingJob.Name, latestRollingJob.Namespace, latestRollingJob.Status)
		if err = r.client.Status().Update(cxt, rolling); err == nil {
			break
		} else {
			klog.Errorf("Failed to update rolling instance %s, %++v", rolling.Name, err)
		}
	}
	return err
}

func (r *ReconcileRolling) getNodes(cxt context.Context, rolling *acv1.Rolling) ([]*v1.Node, error) {
	result := []*v1.Node{}
	nodes := &v1.NodeList{}

	listOptions := &client.ListOptions{}
	if rolling.Spec.NodeSelector.Labels != nil {
		labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: rolling.Spec.NodeSelector.Labels,
		})
		if err != nil {
			return result, err
		}
		listOptions.LabelSelector = labelSelector
	}

	if err := r.client.List(cxt, nodes, listOptions); err != nil {
		return result, err
	}
	for i, _ := range nodes.Items {
		node := &nodes.Items[i]
		// skip master node
		if utils.NodeIsMaster(node) {
			continue
		}
		result = append(result, node)
	}
	return result, nil
}

func labelSelector(rolling *acv1.Rolling) (labels.Selector, error) {
	labelSelector := &metav1.LabelSelector{
		MatchLabels: rollingLabels(rolling),
	}
	return metav1.LabelSelectorAsSelector(labelSelector)
}

func rollingLabels(rolling *acv1.Rolling) map[string]string {
	return map[string]string{
		"rolling":        rolling.Name,
		"controller-uid": string(rolling.UID),
	}
}
