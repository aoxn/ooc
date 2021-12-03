package task

import (
	"fmt"
	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils/hash"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

func (r *ReconcileTask) reinitialize(task *acv1.Task, node *corev1.Node) (reconcile.Result, error) {
	var err error

	hasho, err := hash.HashObject(task.Spec.ConfigTpl)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("hash task: %s", err.Error())
	}
	if hasho == rollingHash(node) {
		klog.Infof("hash does not change, skip reconcile, task=%s, node=%s", hasho, rollingHash(node))
		return reconcile.Result{}, PatchStatus(r.client, task, acv1.PhaseCompleted, "HashNotChanged Skip")
	}

	for k, v := range NewSteps() {
		step := fmt.Sprintf("Step%d", k)
		if !stepFinished(task, step) {
			klog.Infof("reconcile task, run step: %s/%s", step, v.Description)
			result, err := v.Action(r, task, node)
			if err != nil {
				// record failed event
				r.recd.Event(node, corev1.EventTypeWarning, "NodeReinitialize.ReplaceSystemDisk.Failed", err.Error())
				if result.RequeueAfter > 0 {
					klog.Errorf("action failed: requeue after %ds, %s", result.RequeueAfter/time.Second, err.Error())
					return result, nil
				}
				if !result.Requeue {
					klog.Errorf("run steps with error: no requeue, %s", err.Error())
					return result, PatchStatus(r.client, task, acv1.PhaseFailed, err.Error())
				}
				return result, err
			}
			progress := acv1.Progress{
				Step:        step,
				Description: v.Description,
			}
			err = PatchProgress(r.client, task, progress)
			if err != nil {
				klog.Errorf("patch progress status: %s", err.Error())
				return reconcile.Result{}, err
			}
			klog.Infof("progress status patched: %v", progress)
		} else {
			// step has finished before, skip
			klog.Infof("reconcile task, skip finished task %s: %s", step, v.Description)
		}
	}
	r.recd.Event(node, corev1.EventTypeNormal, "NodeReinitialize.ReplaceSystemDisk.Succeed", node.Name)
	klog.Info("finished reconcile: ", task.Namespace, task.Name)
	return reconcile.Result{}, PatchStatus(r.client, task, acv1.PhaseCompleted, "")
}
