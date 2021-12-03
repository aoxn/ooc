package noderepair

import (
	"context"
	"github.com/aoxn/ovm/pkg/context/shared"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/operator/controllers/heal"
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	gerr "github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func AddNodeRepair(
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
	recon := &ReconcileAutoRepair{
		prvd: ctx.ProvdIAAS(),
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		record: mgr.GetEventRecorderFor("AutoHeal"),
	}
	return recon
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"noderepair-controller", mgr,
		controller.Options{
			Reconciler:              r,
			MaxConcurrentReconciles: 1,
		},
	)
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AutoHeal
	return c.Watch(
		&source.Kind{
			Type: &v1.Node{},
		},
		&handler.EnqueueRequestForObject{},
	)
}

// blank assignment to verify that ReconcileAutoRepair implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAutoRepair{}

// ReconcileAutoRepair reconciles a AutoHeal object
type ReconcileAutoRepair struct {
	prvd provider.Interface
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	//record event recorder
	record record.EventRecorder
}


func (r *ReconcileAutoRepair) Reconcile(
	ctx context.Context, request reconcile.Request,
) (reconcile.Result, error) {
	klog.Infof("watch node event: %s", request.Name)
	node := &v1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, node)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return h.NewDelay(5), err
	}

	if h.IsMaster(node) {
		return reconcile.Result{}, nil
	}

	if h.NodeReady(node) {
		return reconcile.Result{}, nil
	}

	return r.fixKubeletNotReady(node)
}


func (r *ReconcileAutoRepair) fixKubeletNotReady(node *v1.Node) (reconcile.Result, error) {
	id := strings.Split(node.Spec.ProviderID, ".")
	if len(id) != 2 {
		klog.Warningf("unrecognized providerid: [%s]", node.Spec.ProviderID)
		return reconcile.Result{}, nil
	}
	instances, err := r.prvd.InstanceDetail(provider.NewEmptyContext(), []string{id[1]})
	if err != nil || len(instances) != 1{
		klog.Warningf("node corresponded ecs not found: [%s], %v", node.Spec.ProviderID, err)
		return reconcile.Result{}, nil
	}
	spec, err := h.MyCluster(r.client)
	if err != nil {
		return reconcile.Result{}, gerr.Wrapf(err, "fix kubelet node")
	}
	klog.Warningf("kubelet not ready: %s, trying to fix", node.Spec.ProviderID)
	reset := heal.NewResteNode(instances[0],spec,node, r.prvd,r.record, r.client)
	err = reset.Execute()
	if err != nil {
		retry,ok := err.(*heal.Retry)
		if ok {
			return h.NewDelay(int64(retry.Duration/time.Second)), err
		}
	}
	return reconcile.Result{}, nil
}

