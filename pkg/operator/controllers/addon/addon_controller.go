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

package addon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aoxn/ooc/pkg/actions/post/addons"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/context/shared"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/utils/kubeclient"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
	//nodepoolv1 "gitlab.alibaba-inc.com/cos/ooc/api/v1"
)

// Add creates a new Rolling Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, ctx *shared.SharedOperatorContext) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
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
	//cauth := ecs.NewClientAuth()
	//err = cauth.Start(ecs.RefreshToken)
	//if err != nil {
	//	panic(fmt.Sprintf("can not connect to ecs provider: %s", err.Error()))
	//}
	return &AddonReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		drain:  drainer,
		//prvd:   ecs.NewProvider(cauth.ECS),
		recd: mgr.GetEventRecorderFor("addon-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"addon-controller", mgr,
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
			Type: &api.Cluster{},
		},
		&handler.EnqueueRequestForObject{},
	)
}

// blank assignment to verify that ReconcileRolling implements reconcile.Reconciler
var _ reconcile.Reconciler = &AddonReconciler{}

// AddonReconciler reconciles a NodePool object
type AddonReconciler struct {
	drain *drain.Helper
	//prvd provider for ecs
	prvd provider.Interface
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	// recd is event record
	recd record.EventRecorder
}

// +kubebuilder:rbac:groups=nodepool.alibaba.com,resources=nodepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nodepool.alibaba.com,resources=nodepools/status,verbs=get;update;patch

func (r *AddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//klog.Infof("cluster for addon reconcile request recieved: %s", req)

	cluster := &api.Cluster{}
	err := r.client.Get(context.TODO(), req.NamespacedName, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			klog.Infof("cluster %s not found, might be delete option, do nothing.", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if cluster.Spec.AddonInitialized {
		return ctrl.Result{}, nil
	}
	if cluster.Spec.Endpoint.Intranet == "" {

		return ctrl.Result{}, fmt.Errorf("cluster intranet slb endpoint is not initialized, wait next retry")
	}
	adds, err := addons.DefaultAddons(&cluster.Spec)
	if err != nil {
		return ctrl.Result{}, err
	}
	var addonList []string
	for k, add := range adds {
		err := kubeclient.ApplyInCluster(add)
		if err != nil {
			klog.Warningf("apply addon[%s] in cluster: %s", k, err.Error())
			return ctrl.Result{}, err
		}
		addonList = append(addonList, k)
	}
	klog.Infof("addons applied: %s", addonList)
	return ctrl.Result{}, patchAddonPhase(r.client, cluster)
}

func patchAddonPhase(
	kcli client.Client,
	spec *api.Cluster,
) error {
	ospec := &api.Cluster{}
	err := kcli.Get(
		context.TODO(),
		client.ObjectKey{Name: spec.Name}, ospec,
	)
	if err != nil {
		return fmt.Errorf("load bootcfg from apiserver: %s", err.Error())
	}
	nspec := ospec.DeepCopy()
	nspec.Spec.AddonInitialized = true

	oldData, err := json.Marshal(ospec)
	if err != nil {
		return fmt.Errorf("marshal ospec: %s", err.Error())
	}
	newData, err := json.Marshal(nspec)
	if err != nil {
		return fmt.Errorf("marshal nspec: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, nspec)
	if patchErr != nil {
		return fmt.Errorf("create merge patch: %s", patchErr.Error())
	}
	return kcli.Patch(
		context.TODO(), nspec,
		client.RawPatch(types.MergePatchType, patchBytes),
	)
}
