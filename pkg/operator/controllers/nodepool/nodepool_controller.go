package nodepool

import (
	"context"
	"fmt"
	acv1 "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/context/shared"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/iaas/provider/alibaba"
	"github.com/aoxn/ovm/pkg/operator/controllers/heal"
	"github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/aoxn/ovm/pkg/utils/hash"
	gerr "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/drain"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

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

var _ reconcile.Reconciler = &ReconcileNodePool{}

// ReconcileNodePool reconciles a Task object
type ReconcileNodePool struct {
	drain  *drain.Helper
	prvd   provider.Interface
	client client.Client
	scheme *runtime.Scheme
	recd   record.EventRecorder

	sharedCtx *shared.SharedOperatorContext
}

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
	// trying to fix nodepool first
	_ = r.repairNodePool(np)

	hasho, err := hash.HashObject(np.Spec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("hash np: %s", err.Error())
	}

	if np.Spec.Infra.Bind == nil {
		klog.Infof("trying to create node pool: %s", np.Name)
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
		klog.Infof("create nodepool finished: %s", np.Name)
		err = help.Patch(r.client, np, diff, help.PatchSpec)
		if err != nil {
			return help.NewDelay(3), gerr.Wrap(err, "patch bind infra")
		}
	} else {
		if hasho == nodePoolHash(np) {
			klog.Infof("hash does not change, "+
				"skip reconcile, np=%s, node=%s", hasho, nodePoolHash(np))

			return reconcile.Result{}, nil
		}

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
	klog.Infof("wait for nodepool[%s] replicas finished: %d", np.Name, np.Spec.Infra.DesiredCapacity)
	defer klog.Infof("wait for nodepool[%s] replicas finished", np.Name)
	return reconcile.Result{}, WaitReplicas(np.Spec.Infra.DesiredCapacity)
}

func (r *ReconcileNodePool) repairNodePool(pool *acv1.NodePool) error {
	bind := pool.Spec.Infra.Bind
	if bind == nil {
		return nil
	}

	detail, err := r.prvd.ScalingGroupDetail(
		r.sharedCtx.ProviderCtx(), bind.ScalingGroupId,
		provider.Option{Action: alibaba.ActionInstanceIDS},
	)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidVPC") {
			diff := func(copy runtime.Object) (client.Object, error) {
				np := copy.(*acv1.NodePool)
				np.Spec.Infra.Bind = nil
				pool.Spec.Infra.Bind = nil
				return np, nil
			}
			klog.Warningf("clear nodepool bind infra for invalid vpc: [%s], [%s]", pool.Name, pool.Spec.Infra.Bind)
			err = help.Patch(r.client, pool, diff, help.PatchSpec)
			if err != nil {
				klog.Warningf("clear nodepool bind infra for invalid vpc, %s, %s", pool.Name, err.Error())
			}
		}
		return gerr.Wrapf(err, "group %s detail", bind.ScalingGroupId)
	}
	nodes, err := help.Workers(r.client)
	if err != nil {
		return gerr.Wrapf(err, "list nodes")
	}
	var miss []provider.Instance
	for _, v := range detail.Instances {
		found := false
		for _, n := range nodes {
			if strings.Contains(n.Spec.ProviderID, v.Id) {
				found = true
				break
			}
		}
		if !found {
			miss = append(miss, v)
		}
	}
	spec, err := help.MyCluster(r.client)
	if err != nil {
		return gerr.Wrapf(err, "find cluster spec")
	}
	for _, m := range miss {
		klog.Infof("instance %s.%s miss node object, try to reset node", m.Id, m.Ip)
		err = heal.NewResteNode(
			m, spec, nil, r.prvd, r.recd, r.client,
		).Execute()
		if err != nil {
			klog.Errorf("reset node: %s", err.Error())
		}
		time.Sleep(10 * time.Second)
	}

	//
	var extra []v1.Node
	for _, n := range nodes {
		found := false
		for _, in := range detail.Instances {
			if strings.Contains(n.Spec.ProviderID, in.Id) {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, n)
		}
	}
	for _, n := range extra {
		klog.Warningf("try to delete extra node[%s]", n.Name)
		// remove
		err := r.client.Delete(context.TODO(),&n)
		if err != nil {
			klog.Errorf("delete extra node[%s] fail", n.Name)
		}
	}
	return nil
}

func WaitReplicas(i int) error {
	return nil
}

func nodePoolHash(node *acv1.NodePool) string {
	lbl := node.GetLabels()
	if lbl == nil {
		return ""
	}
	return lbl[acv1.NodePoolHashLabel]
}
