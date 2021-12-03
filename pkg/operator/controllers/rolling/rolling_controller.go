package rolling

import (
	"context"
	"github.com/aoxn/ooc/pkg/context/shared"
	"k8s.io/klog/v2"

	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Rolling Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddRollingController(
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
	recorder := mgr.GetEventRecorderFor("rolling-controller")
	return &ReconcileRolling{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: recorder}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rolling-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Rolling
	err = c.Watch(&source.Kind{Type: &acv1.Rolling{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	hand := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &acv1.Rolling{},
	}
	// Watch for changes to secondary resource Pods and requeue the owner Rolling
	return c.Watch(
		&source.Kind{Type: &acv1.Task{}},
		&EnqueueTaskRequest{
			EnqueueRequestForOwner: hand,
		},
	)
}

// blank assignment to verify that ReconcileRolling implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRolling{}

// ReconcileRolling reconciles a Rolling object
type ReconcileRolling struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Rolling object and makes changes based on the state read
// and what is in the Rolling.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRolling) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	klog.Infof("Reconciling Rolling, %s, %s", request.Name, request.Namespace)

	// Fetch the Rolling instance
	instance := &acv1.Rolling{}
	cxt := context.TODO()
	err := r.client.Get(cxt, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err := r.manage(cxt, instance); err != nil {
		klog.Errorf("failed to manage rolling: %s, err: %++v", instance.Name, err)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
