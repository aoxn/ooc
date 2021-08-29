package task

import (
	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

type Step struct {
	Description string
	Action      func(r *ReconcileTask, task *acv1.Task, node *v1.Node) (reconcile.Result, error)
}

func NewSteps() []Step {
	return []Step{
		{
			Description: "DrainNode",
			Action:      drainNode,
		},
		{
			Description: "ReplaceSystemDisk",
			Action:      replaceSystemDisk,
		},
		{
			Description: "UnCordon",
			Action:      uncordon,
		},
	}
}

func drainNode(r *ReconcileTask, task *acv1.Task, node *v1.Node) (reconcile.Result, error) {
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: 20 * time.Second,
	}, WaitDrainNode(r.drain, node)
}

func replaceSystemDisk(r *ReconcileTask, task *acv1.Task, node *v1.Node) (reconcile.Result, error) {
	// Add retry here. for 3 times most.
	// Fail Task when error times reached limit. Because we dont want
	// to replace system disk again and again which would do no good to
	// recovery
	id := strings.Split(node.Spec.ProviderID, ".")
	err := wait.ExponentialBackoff(
		wait.Backoff{
			Factor:   2,
			Steps:    3,
			Duration: 5 * time.Second,
		},
		func() (done bool, err error) {
			err = r.prvd.ReplaceSystemDisk(r.sharedCtx.ProviderCtx(), id[1], "", provider.Option{})
			if err != nil {
				klog.Infof("replace system disk: %s, requeue after %d minute", err.Error(), 1)
				return false, nil
			}
			err = WaitForNodeReady(r.drain.Client, task, node.Name)
			if err != nil {
				return false, nil
			}
			return true, nil
		},
	)

	return reconcile.Result{Requeue: false}, err
}

func uncordon(r *ReconcileTask, task *acv1.Task, node *v1.Node) (reconcile.Result, error) {
	return reconcile.Result{}, Cordon(r.drain, node, false)
}
