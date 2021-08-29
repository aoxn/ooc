package nodepool

import (
	"context"
	"fmt"
	"github.com/aoxn/ooc/pkg/context/shared"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/operator/controllers/help"
	"github.com/aoxn/ooc/pkg/utils/hash"
	gerr "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"time"

	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
func AddNodePoolController(
	mgr manager.Manager,
	ctx *shared.SharedOperatorContext,
) error {
	return add(mgr, newReconciler(mgr, ctx))
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

	return &ReconcileNodePool{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		sharedCtx: ctx,
		prvd:      ctx.ProvdIAAS(),
		recd:      mgr.GetEventRecorderFor("task-controller"),
	}
}

const NODE_POOL_FINALIZER = "nodepool"

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"nodepool-controller", mgr,
		controller.Options{
			Reconciler:              r,
			MaxConcurrentReconciles: 2,
		},
	)
	if err != nil {
		return fmt.Errorf("create task controller: %s", err.Error())
	}

	// Watch for changes to primary resource Task
	return c.Watch(
		&source.Kind{
			Type: &acv1.NodePool{},
		},
		&handler.EnqueueRequestForObject{},
	)
}

// blank assignment to verify that ReconcileNodePool implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodePool{}

// ReconcileNodePool reconciles a Task object
type ReconcileNodePool struct {
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
func (r *ReconcileNodePool) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("reconcile NodePool: %s", request.NamespacedName)
	np := &acv1.NodePool{}
	mctx := r.sharedCtx.ProviderCtx()
	err := r.client.Get(context.TODO(), request.NamespacedName, np)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("nodepool %s not found, do delete", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		return help.NewDelay(3), err
	}

	//spec, err := MyCluster(r.client)
	//if err != nil {
	//	return reconcile.Result{}, gerr.Wrap(err, "find cluster")
	//}
	//fmt.Printf("%s\n",hash.PrettyYaml(np))
	if !np.DeletionTimestamp.IsZero() {
		klog.Infof("nodepool has been deleted, [%s], %v", np.Name, np.Spec.Infra.Bind)
		err := r.prvd.DeleteNodeGroup(mctx, np)
		if err != nil {
			return help.NewDelay(10), gerr.Wrapf(err, "clean up nodepool infra", np.Name)
		}
		diff := func(copy runtime.Object) (client.Object, error) {
			mp := copy.(*acv1.NodePool)
			mp.Finalizers = help.Remove(mp.Finalizers, NODE_POOL_FINALIZER)
			return mp, nil
		}
		return reconcile.Result{}, help.Patch(r.client, np, diff, help.PatchSpec)
	}
	hasho, err := hash.HashObject(np.Spec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("hash np: %s", err.Error())
	}
	if hasho == nodePoolHash(np) {
		klog.Infof("hash does not change, "+
			"skip reconcile, np=%s, node=%s", hasho, nodePoolHash(np))

		return reconcile.Result{}, nil
	}

	if np.Spec.Infra.Bind == nil {
		bind, err := r.prvd.CreateNodeGroup(mctx, np)
		if err != nil {
			return reconcile.Result{}, gerr.Wrapf(err, "create node group: %v", np.Name)
		}
		diff := func(copy runtime.Object) (client.Object, error) {
			mp := copy.(*acv1.NodePool)
			mp.Spec.Infra.Bind = bind
			if !help.Has(mp.Finalizers, NODE_POOL_FINALIZER) {
				mp.Finalizers = append(mp.Finalizers, NODE_POOL_FINALIZER)
			}
			return mp, nil
		}
		err = help.Patch(r.client, np, diff, help.PatchSpec)
		if err != nil {
			return help.NewDelay(3), gerr.Wrap(err, "patch bind infra")
		}
	} else {
		klog.Infof("node group modified, %s", np.Name)
		err := r.prvd.ModifyNodeGroup(mctx, np)
		if err != nil {
			return help.NewDelay(3), gerr.Wrapf(err, "modify node group: %v", np)
		}
	}
	diff := func(copy runtime.Object) (client.Object, error) {
		np := copy.(*acv1.NodePool)
		if np.Labels == nil {
			np.Labels = map[string]string{}
		}
		np.Labels[acv1.NodePoolHashLabel] = hasho
		return np, nil
	}
	err = help.Patch(r.client, np, diff, help.PatchSpec)
	if err != nil {
		klog.Warningf("patch nodepool hash label fail, %s, %s", np.Name, err.Error())
	}
	return reconcile.Result{}, WaitReplicas(np.Spec.Infra.DesiredCapacity)
}

func WaitReplicas(i int) error {
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

func nodePoolHash(node *acv1.NodePool) string {
	lbl := node.GetLabels()
	if lbl == nil {
		return ""
	}
	return lbl[acv1.NodePoolHashLabel]
}
