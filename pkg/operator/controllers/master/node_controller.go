/*
Copyright 2020 aoxn.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package master

import (
	"context"
	"fmt"
	ctx "github.com/aoxn/ovm/pkg/context"
	"github.com/aoxn/ovm/pkg/context/shared"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/operator/controllers/heal"
	"github.com/aoxn/ovm/pkg/operator/controllers/help"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/drain"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//nodepoolv1 "gitlab.alibaba-inc.com/cos/ovm/api/v1"
)

// Add creates a new Rolling Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddNode(
	mgr manager.Manager,
	ctx *shared.SharedOperatorContext,
) error {
	return add(mgr, newRepairReconciler(mgr, ctx))
}

// newMasterSetReconciler returns a new reconcile.Reconciler
func newRepairReconciler(
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

	return &RepairReconciler{
		cache:  ctx.NodeCacheCtx(),
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		drain:  drainer,
		heal:   ctx.MemberHeal(),
		prvd:   ctx.ProvdIAAS(),
		recd:   mgr.GetEventRecorderFor("task-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"master-controller", mgr,
		controller.Options{
			Reconciler:              r,
			MaxConcurrentReconciles: 1,
		},
	)
	if err != nil {
		return fmt.Errorf("create task controller: %s", err.Error())
	}

	// Watch for changes to primary resource Task
	return c.Watch(
		&source.Kind{
			Type: &v1.Node{},
		},
		&handler.EnqueueRequestForObject{},
	)
}

// blank assignment to verify that ReconcileRolling implements reconcile.Reconciler
var _ reconcile.Reconciler = &RepairReconciler{}

// MasterSetReconciler reconciles a NodePool object
type RepairReconciler struct {
	heal  *heal.MasterHeal
	drain *drain.Helper
	//prvd provider for ecs
	prvd provider.Interface
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	// recd is event record
	recd record.EventRecorder

	// master context cache
	cache *ctx.CachedContext
}

func (r *RepairReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	n, err := help.Node(r.client, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			// nothing to do
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if !help.IsMaster(n) {
		// do not process non-master node
		return ctrl.Result{}, err
	}
	if !help.NodeReady(n) {
		klog.Infof("master node not ready, try meber heal: %s", req.Name)
		done := make(chan struct{})
		r.heal.NotifyNodeEvent(done)
		res := <-done
		klog.Infof("master repair done: %v", res)
	}
	return ctrl.Result{}, nil
}
