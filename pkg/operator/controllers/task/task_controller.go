package task

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aoxn/ooc/pkg/context/shared"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/aoxn/ooc/pkg/utils/hash"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"os"
	"sort"
	"strings"
	"time"

	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	gerror "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/kubectl/pkg/drain"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Task Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddTaskController(
	mgr manager.Manager,
	ctx *shared.SharedOperatorContext,
) error {
	return add(mgr, newReconciler(mgr,ctx))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(
	mgr manager.Manager,
	ctx *shared.SharedOperatorContext,
) reconcile.Reconciler {
	mclient, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		panic(fmt.Sprintf("create client: %s", mclient))
	}
	drainer := &drain.Helper{
		Timeout:                         15 * time.Minute,
		SkipWaitForDeleteTimeoutSeconds: 60,
		Client:                          mclient,
		GracePeriodSeconds:              -1,
		DisableEviction:                 false,
		IgnoreAllDaemonSets:             true,
		Force:                           true,
		Out:                             os.Stdout,
		ErrOut:                          os.Stderr,
	}

	return &ReconcileTask{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		drain:  drainer,
		sharedCtx: ctx,
		prvd:   ctx.ProvdIAAS(),
		recd:   mgr.GetEventRecorderFor("task-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"task-controller", mgr,
		controller.Options{
			Reconciler:              r,
			MaxConcurrentReconciles: 30,
		},
	)
	if err != nil {
		return fmt.Errorf("create task controller: %s", err.Error())
	}

	// Watch for changes to primary resource Task
	return c.Watch(
		&source.Kind{
			Type: &acv1.Task{},
		},
		&handler.EnqueueRequestForObject{},
	)
}

// blank assignment to verify that ReconcileTask implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTask{}

// ReconcileTask reconciles a Task object
type ReconcileTask struct {
	drain *drain.Helper
	//prvd provider for ecs
	prvd provider.Interface
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	// recd is event record
	recd record.EventRecorder

	sharedCtx *shared.SharedOperatorContext
}

// Reconcile reads that state of the cluster for a Task object and makes changes based on the state read
// and what is in the Task.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTask) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("reconcile task: %s", request.NamespacedName)
	// Fetch the Task task
	task := &acv1.Task{}
	err := r.client.Get(context.TODO(), request.NamespacedName, task)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.Infof("task %s not found, might be delete option, do nothing.", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if task.Status.Phase == acv1.PhaseCompleted ||
		task.Status.Phase == acv1.PhaseFailed {
		klog.Infof("task already [%s], return immediately", task.Status.Phase)
		return reconcile.Result{}, nil
	}

	if task.Spec.NodeName == "" {
		// NodeName must be provided, error immediately
		return reconcile.Result{
			Requeue: false,
		}, PatchStatus(r.client, task, acv1.PhaseFailed, "task.Spec.Name must be provided")
	}
	node := &v1.Node{}
	err = r.client.Get(
		context.TODO(), client.ObjectKey{Name: task.Spec.NodeName}, node,
	)
	if err != nil {
		// not found? error immediately
		return reconcile.Result{}, fmt.Errorf("find node: %s", err.Error())
	}

	klog.Infof("receieve %s task, dispatch", task.Spec.TaskType)
	switch task.Spec.TaskType {
	case acv1.TaskTypeUpgrade:
		return r.reinitialize(task, node)
	case acv1.TaskTypePod:
		return reconcile.Result{}, fmt.Errorf("unimplemented")
	case acv1.TaskTypeAutoRepair:
		// do double check for kubelet ready
		notReady, reason := utils.KubeletNotReady(node)
		if notReady {
			klog.Infof("kubelet not ready, do repair: %s", reason)
			return r.doAutoRepair(task, node)
		}
		klog.Infof("kubelet is in ready status, skip repair")
		return reconcile.Result{}, PatchStatus(r.client, task, acv1.PhaseCompleted, "NoNeedRepair")
	case acv1.TaskTypeCommand:
		return reconcile.Result{}, fmt.Errorf("unimplemented")
	}
	klog.Infof("unknown task type: [%s], return immediately", task.Spec.TaskType)
	// unknown task type, error immediately
	return reconcile.Result{}, nil
}

func WaitDrainNode(dr *drain.Helper, node *v1.Node) error {
	klog.Infof("try cordon node")
	err := Cordon(dr, node, true)
	if err != nil {
		return fmt.Errorf("cordon node: %s", err.Error())
	}
	pods, errs := dr.GetPodsForDeletion(node.Name)
	if errs != nil {
		return gerror.NewAggregate(errs)
	}
	warnings := pods.Warnings()
	if warnings != "" {
		klog.Infof("WARNING: drain %s", warnings)
	}
	err = dr.DeleteOrEvictPods(pods.Pods())
	if err != nil {
		pending, newErrs := dr.GetPodsForDeletion(node.Name)
		if pending != nil {
			pods := pending.Pods()
			if len(pods) != 0 {
				klog.Infof("there are pending pods in node %q when an error occurred: %v", node.Name, err)
				for _, pod := range pods {
					klog.Infof("pods: %s/%s", pod.Namespace, pod.Name)
				}
			}
		}
		if newErrs != nil {
			klog.Infof("drain: GetPodsForDeletion, %s", gerror.NewAggregate(newErrs))
		}
		return err
	}
	klog.Info("drain finished")
	return nil
}

func Cordon(dr *drain.Helper, node *v1.Node, cordon bool) error {
	help := drain.NewCordonHelper(node)
	help.UpdateIfRequired(cordon)
	err, patcherr := help.PatchOrReplace(dr.Client, false)
	if err != nil || patcherr != nil {
		return gerror.NewAggregate([]error{err, patcherr})
	}
	return nil
}

func WaitForNodeReady(
	client kubernetes.Interface,
	task *acv1.Task,
	nodeName string,
) error {
	klog.Infof("wait for node ready")
	waitReady := func() (done bool, err error) {
		node, err := client.CoreV1().Nodes().Get(
			context.TODO(), nodeName,
			metav1.GetOptions{},
		)
		if err != nil {
			klog.Infof("wait node ready: %s", err.Error())
			return false, nil
		}
		var ready *v1.NodeCondition
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady {
				ready = &condition
				break
			}
		}
		if ready == nil ||
			ready.Status == v1.ConditionFalse ||
			ready.Status == v1.ConditionUnknown {
			klog.Infof("kubelet not ready yet: %s", nodeName)
			return false, nil
		}
		// node status heartbeat time should no later then task create time
		// this ensures that NodeStatus flipped after replace system disk.
		if ready.LastHeartbeatTime.Before(&task.CreationTimestamp) {
			klog.Infof("node ready but heartbeat time is not updated. wait for heartbeat update")
			return false, nil
		}

		// ensure kubelet version has been syncd
		if node.Status.NodeInfo.KubeletVersion !=
			task.Spec.ConfigTpl.Kubernetes.Version {
			klog.Infof("kubelet version does not equal: node=%s, expected=%s",
				node.Status.NodeInfo.KubeletVersion, task.Spec.ConfigTpl.Kubernetes.Version)
			return false, nil
		}
		klog.Infof("kubelet ready")
		return true, nil
	}
	return wait.PollImmediate(
		10*time.Second, 10*time.Minute,
		func() (done bool, err error) {
			// wait for node status steady in 6s
			for i := 0; i < 3; i++ {
				ready, err := waitReady()
				if err != nil || !ready {
					return ready, err
				}
				klog.Infof("node[%s] status ready, wait for status steady[%d]", nodeName, i)
				time.Sleep(2 * time.Second)
			}
			return true, nil
		},
	)
}

func PatchProgress(
	mclient client.Client,
	task *acv1.Task,
	progress acv1.Progress,
) error {

	otask := &acv1.Task{}
	err := mclient.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      task.Name,
			Namespace: task.Namespace,
		}, otask,
	)
	if err != nil {
		return fmt.Errorf("get task: %s", err.Error())
	}
	ntask := otask.DeepCopy()
	found := false
	for i := range ntask.Status.Progress {
		p := ntask.Status.Progress[i]
		if p.Step == progress.Step {
			found = true
			if p.Description != progress.Description {
				p.Description = progress.Description
				break
			} else {
				klog.Infof("patch progress: skip %s", p.Step)
				return nil
			}
		}
	}
	if !found {
		ntask.Status.Progress = append(ntask.Status.Progress, progress)
	}

	sort.SliceStable(
		ntask.Status.Progress,
		func(i, j int) bool {
			return ntask.Status.Progress[i].Step < ntask.Status.Progress[j].Step
		},
	)

	oldData, err := json.Marshal(otask)
	if err != nil {
		return fmt.Errorf("marshal otask: %s", err.Error())
	}
	newData, err := json.Marshal(ntask)
	if err != nil {
		return fmt.Errorf("marshal ntask: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, otask)
	if patchErr != nil {
		return fmt.Errorf("create patch: %s", err.Error())
	}
	klog.Infof("try patch progress status: %v", progress)
	return mclient.Status().Patch(
		context.TODO(), ntask, client.RawPatch(types.MergePatchType, patchBytes),
	)
}

// PatchStatus
// patch Task status. 1). task.status.phase 2). task.status.hash
// patch Node label   1). node.labels.[hash]
func PatchStatus(
	mclient client.Client,
	task *acv1.Task,
	phase, reason string,
) error {
	otask := &acv1.Task{}
	err := mclient.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      task.Name,
			Namespace: task.Namespace,
		}, otask,
	)
	if err != nil {
		return fmt.Errorf("get task: %s", err.Error())
	}
	ntask := otask.DeepCopy()
	ntask.Status.Phase = phase
	ntask.Status.Reason = reason
	if phase == acv1.PhaseCompleted {
		value, err := hash.HashObject(task.Spec.ConfigTpl)
		if err != nil {
			return fmt.Errorf("compute hash: %s", err.Error())
		}
		ntask.Status.Hash = value
		err = PatchNodeHash(mclient, ntask, value)
		if err != nil {
			klog.Warningf("patch node hash error: %s", err.Error())
		}
	}
	oldData, err := json.Marshal(otask)
	if err != nil {
		return fmt.Errorf("marshal otask: %s", err.Error())
	}
	newData, err := json.Marshal(ntask)
	if err != nil {
		return fmt.Errorf("marshal ntask: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, otask)
	if patchErr != nil {
		return fmt.Errorf("create merge patch: %s", err.Error())
	}
	return mclient.Status().Patch(
		context.TODO(), ntask,
		client.RawPatch(types.MergePatchType, patchBytes),
	)
}

func PatchNodeHash(
	mclient client.Client,
	task *acv1.Task,
	hash string,
) error {
	onode := &v1.Node{}
	err := mclient.Get(
		context.TODO(),
		client.ObjectKey{
			Name: task.Spec.NodeName,
		}, onode,
	)
	if err != nil {
		return fmt.Errorf("get node: %s", err.Error())
	}
	nnode := onode.DeepCopy()
	nnode.Labels[acv1.RollingHashLabel] = hash

	oldData, err := json.Marshal(onode)
	if err != nil {
		return fmt.Errorf("marshal onode: %s", err.Error())
	}
	newData, err := json.Marshal(nnode)
	if err != nil {
		return fmt.Errorf("marshal nnode: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, onode)
	if patchErr != nil {
		return fmt.Errorf("create merge patch: %s", patchErr.Error())
	}
	return mclient.Status().Patch(
		context.TODO(), nnode,
		client.RawPatch(types.MergePatchType, patchBytes),
	)
}

func stepFinished(
	task *acv1.Task,
	step string,
) bool {
	for _, progress := range task.Status.Progress {
		if progress.Step == step {
			return true
		}
	}
	return false
}

func rollingHash(node *v1.Node) string {
	lbl := node.GetLabels()
	if lbl == nil {
		return ""
	}
	return lbl[acv1.RollingHashLabel]
}

func nodeFromProviderID(id string) (string, string, error) {
	tmp := strings.Split(id, ".")
	if len(tmp) != 2 {
		return "", "", fmt.Errorf("unknown: %s, format ${region}.${instanceid}", id)
	}
	return tmp[0], tmp[1], nil
}
