package autorepair

import (
	"context"
	"fmt"
	"github.com/aoxn/ooc/pkg/context/shared"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	h "github.com/aoxn/ooc/pkg/operator/controllers/help"
	"github.com/aoxn/ooc/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"time"

	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	corev1 "k8s.io/api/core/v1"
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

const (
	TimeStamp = "alibabacloud.com/started.timestamp"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new AutoHeal Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddAutoRepairController(
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
	recon := &ReconcileAutoRepair{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		record: mgr.GetEventRecorderFor("AutoHeal"),
	}
	go recon.CleanUp()
	return recon
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"autorepair-controller", mgr,
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
			Type: &corev1.Node{},
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

func (r *ReconcileAutoRepair) CleanUp() {
	clean := func() {
		klog.Infof("clean up completed task.")
		tasks := &acv1.TaskList{}
		err := r.client.List(context.TODO(), tasks)
		if err != nil {
			klog.Infof("clean up completed task: list %s, retry in next 10 minutes", err.Error())
			return
		}
		for _, task := range tasks.Items {
			if task.Spec.TaskType != acv1.TaskTypeAutoRepair {
				continue
			}
			if task.Status.Phase == acv1.PhaseCompleted {
				err = r.client.Delete(context.TODO(), &task)
				if err != nil {
					time.Sleep(2 * time.Second)
					klog.Infof("clean up completed task: delete fail,"+
						" %s, retry in next 10 minutes. %s", err.Error(), task.Name)
				} else {
					klog.Infof("clean up completed task: %s", task.Name)
				}
			}
		}
	}
	wait.Until(
		clean, 10*time.Minute, make(chan struct{}),
	)
}

// Reconcile reads that state of the cluster for a AutoHeal object and makes changes based on the state read
// and what is in the AutoHeal.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAutoRepair) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	//rlog.Info("try reconcile repair: %s", request)

	// Fetch the AutoHeal node
	node := &corev1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, node)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("node not found, skip")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if h.IsMaster(node) {
		// master is not our responsibility
		return reconcile.Result{}, nil
	}
	// control repair scope
	//managed, reason := isManagedNodePool(r.client, node)
	//if ! managed {
	//	rlog.Infof("node is not managed by ManagedNodePool, %s", reason)
	//	return reconcile.Result{}, nil
	//}

	notReady, reason := utils.KubeletNotReady(node)
	if ! notReady {
		// node is ready. return immediately
		return reconcile.Result{}, nil
	}

	klog.Warningf("kubelet not ready: %s, trying to fix", reason)
	return r.fixKubeletNotReady(node)
}

func (r *ReconcileAutoRepair) fixKubeletNotReady(node *corev1.Node) (reconcile.Result, error) {

	// TODO: fix me
	//   make sure no more than 2 task in concurrency.
	//   we are safe for now, because ReconcileAutoRepair concurrency has been set to 1.
	tasks, oin, ofail, err := TaskStatistic(r.client)
	if err != nil {
		return reconcile.Result{}, err
	}
	vtask := newTask(node)
	// 1. can not start new node repair process when all the 2 task failed.
	// 2. failed task can be restarted atfer 3 minutes
	mtask, exist := findTask(tasks, vtask)
	if exist {
		// figure out whether task has failed before.
		// restart after 10 minutes if it does.
		klog.Infof("repair task already exist")
		switch mtask.Status.Phase {
		case acv1.PhaseCompleted:
			klog.Infof("restart a previous completed task immediately")
			return reconcile.Result{}, restartTask(r.client, vtask)
		case acv1.PhaseFailed:
			klog.Infof("task has failed previously, " +
				"wait for %f minutes to restart", RestartInterval/time.Minute)
			if taskNeedRestart(tasks, vtask) {
				klog.Infof("try to restart failed task after " +
					"failing for %f minutes, %s", vtask.Name, RestartInterval/time.Minute)
				return reconcile.Result{}, restartTask(r.client, vtask)
			}
		}
		klog.Infof("task is in progress, wait")
		// task in progress, skip
		return reconcile.Result{}, nil
	} else {
		klog.Infof("repair task does not exist, create new one")
		if ofail > 2 {
			// no more then two task should be reconciled at the same time.
			// especially failed task.
			// ofail >2 means at least two task has failed.
			// skip for another reconcile slot.
			klog.Infof("at least two task has failed. skip for another reconcile slot.")
			return reconcile.Result{}, nil
		}
		if oin > 2 {
			// same as above
			klog.Infof("at least two task is in reconciling, skip for another reconcile slot")
			return reconcile.Result{}, nil
		}
		// kubelet not ready , activate repair
		return reconcile.Result{}, createTask(r.client, vtask)
	}
}

func findTask(
	tasks *acv1.TaskList,
	vtask *acv1.Task,
) (*acv1.Task, bool) {

	for _, task := range tasks.Items {
		if vtask.Name == task.Name {
			return &task, true
		}
	}
	return nil, false
}

func createTask(
	rclient client.Client,
	task *acv1.Task,
) error {
	return rclient.Create(context.TODO(), task)
}

func restartTask(
	rclient client.Client,
	task *acv1.Task,
) error {
	err := rclient.Delete(context.TODO(), task)
	if err != nil {
		return fmt.Errorf("delete task %s", task.Name)
	}
	return rclient.Create(context.TODO(), task)
}

const RestartInterval = 15 * time.Minute

func taskNeedRestart(
	tasks *acv1.TaskList,
	vtask *acv1.Task,
) bool {
	for _, task := range tasks.Items {
		if vtask.Name == task.Name {
			if task.Status.Phase == acv1.PhaseFailed {
				ago := metav1.NewTime(time.Now().Add(-1 * RestartInterval))
				// restart task which failed for 15 minutes
				return task.CreationTimestamp.Before(&ago)
			}
		}
	}
	return false
}

func taskName(name string) string {
	return fmt.Sprintf("%s-autorepair", name)
}

func newTask(node *corev1.Node) *acv1.Task {
	return &acv1.Task{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              taskName(node.Name),
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: acv1.TaskSpec{
			NodeName: node.Name,
			TaskType: acv1.TaskTypeAutoRepair,
			ConfigTpl: acv1.ConfigTpl{
				Kubernetes: acv1.Unit{
					Name:    "kubernetes",
					Version: node.Status.NodeInfo.KubeletVersion,
				},
			},
		},
	}
}

func TaskStatistic(
	rclient client.Client,
) (*acv1.TaskList, int, int, error) {
	tasks := &acv1.TaskList{}
	err := rclient.List(context.TODO(), tasks)
	if err != nil {
		return nil, 0, 0, err
	}
	// oin: task count which not failed or completed
	// ofail: task count which failed
	oin, ofail := 0, 0
	for _, t := range tasks.Items {
		if t.Status.Phase != acv1.PhaseFailed &&
			t.Status.Phase != acv1.PhaseCompleted {
			oin += 1
		}
		if t.Status.Phase == acv1.PhaseFailed {
			ofail += 1
		}
	}
	return tasks, oin, ofail, nil
}

func isManagedNodePool(rclient client.Client, node *corev1.Node) (bool, string) {
	lbl := node.Labels
	if lbl == nil {
		lbl = make(map[string]string)
	}
	id, ok := lbl[acv1.NodePoolIDLabel]
	if !ok {
		// no nodepool id found, default to normal node pool
		return false, "NotManagedNodePool.NodePoolLabelNotFound"
	}
	pool := &acv1.NodePool{}
	err := rclient.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      id,
		}, pool,
	)
	if err != nil {
		return false, fmt.Sprintf("Waring: skip, %s", err.Error())
	}
	return pool.Spec.AutoHeal, ""
}
