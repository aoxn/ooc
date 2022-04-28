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
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/context/shared"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/aoxn/ovm/pkg/operator/heal"
	"github.com/pkg/errors"
	gerr "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

func AddMasterSet(
	mgr manager.Manager,
	ctx *shared.SharedOperatorContext,
) error {
	return addMasterSet(mgr, newMasterSetReconciler(mgr, ctx))
}

// newMasterSetReconciler returns a new reconcile.Reconciler
func newMasterSetReconciler(
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
	//cauth := ecs.NewClientAuth()
	//err = cauth.Start(ecs.RefreshToken)
	//if err != nil {
	//	panic(fmt.Sprintf("can not connect to ecs provider: %s", err.Error()))
	//}
	klog.Infof("initialize master controller")
	return &MasterSetReconciler{
		sharedCtx: ctx,
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		drain:     drainer,
		healet:    ctx.MemberHeal(),
		prvd:      ctx.ProvdIAAS(),
		recd:      mgr.GetEventRecorderFor("task-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func addMasterSet(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(
		"masterset-controller", mgr,
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
			Type: &api.MasterSet{},
		},
		&handler.EnqueueRequestForObject{},
	)
}

// blank assignment to verify that ReconcileRolling implements reconcile.Reconciler
var _ reconcile.Reconciler = &MasterSetReconciler{}

// MasterSetReconciler reconciles a NodePool object
type MasterSetReconciler struct {
	recd   record.EventRecorder
	mlog   log.Logger
	healet *heal.Healet
	drain  *drain.Helper
	prvd   provider.Interface
	client client.Client
	scheme *runtime.Scheme
	stack  map[string]provider.Value

	sharedCtx *shared.SharedOperatorContext
}

func (m *MasterSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	if ! m.healet.Healthy() {
		klog.Infof("master state still in UnHealthy state, trigger repair")
		err := m.healet.FixMasterNode()
		if err != nil {
			klog.Errorf("masterset controller: fix control plane with [%s]", err.Error())
		}
		return ctrl.Result{Requeue: true, RequeueAfter: 1 * time.Minute}, nil
	}

	ms, err := help.MasterSet(m.client, "masterset")
	if err != nil {
		return ctrl.Result{RequeueAfter: 1 * time.Minute, Requeue: true}, nil
	}

	// sharedCtx is initialized on controller start.
	cctx := m.sharedCtx.ProviderCtx()

	// 1. [FastCheck] check replicas equal count(Node[master]).
	//    return immediately on equal
	mnode, err := help.MasterNodes(m.client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get nodes: %s", err.Error())
	}
	if len(mnode) == ms.Spec.Replicas {
		klog.Infof("master node is as expected replica count: %d, nothing to do", len(mnode))
		return ctrl.Result{}, nil
	}

	// 2. [DoubleCheck] check replicas equal count(ECS[master])
	// 		return immediately on equal
	detail, err := m.prvd.ScalingGroupDetail(
		cctx, "", provider.Option{Action: "InstanceIDS"},
	)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: 15 * time.Second}, nil
	}
	if len(detail.Instances) == ms.Spec.Replicas {
		// TODO: how about scale in???
		//   some error might occurred. send repair signal?
		klog.Infof("master ecs is as expected "+
			"replica count: %d, nothing to do", len(detail.Instances))
		return ctrl.Result{}, nil
	}

	// 3. do scale ecs scaling group
	ud, err := m.prvd.UserData(cctx, provider.JoinMasterUserdata)
	if err != nil {
		return ctrl.Result{}, gerr.Wrapf(err, "join master userdata")
	}
	err = m.prvd.ModifyScalingConfig(cctx, "",
		provider.Option{
			Action: "UserData",
			Value:  provider.Value{Val: ud},
		},
	)
	if err != nil {
		klog.Errorf("modify userdata: %s", err.Error())
		return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, nil
	}

	scale := func(expect int) error {

		klog.Infof("[QuorumScale] wait on member center to finish cluster heal")
		if ! m.healet.Healthy(){
			return errors.Wrapf(err, "[QuorumScale] controlplane not healthy, wait retry")
		}

		klog.Infof("[QuorumScale] do scale[expect=%d]...........................", expect)
		// check for etcd member, make sure len(etcd)==len(ecs)
		if expect == 1 {
			klog.Infof("[QuorumScale] do remove ecs to 1")
			// replica changes from 2 to 1.
			ip, err := m.healet.RemoveFollower()
			if err != nil {
				return fmt.Errorf("quorum remove etcd member: %s", err.Error())
			}
			id := findId(detail.Instances, ip)
			if id == "" {
				return fmt.Errorf("ecs not found by ip: %s", ip)
			}
			err = m.prvd.RemoveScalingGroupECS(cctx, "", id)
			if err != nil {
				return fmt.Errorf("remove ecs from scaling group: %s", err.Error())
			}
		} else {
			klog.Infof("[QuorumScale] do scale ecs to %d", expect)
			// replicas changes from n to min(replica,2)
			err = m.prvd.ScaleMasterGroup(cctx, "", expect)
			if err != nil {
				klog.Infof("[QuorumScale] sleep 30s for scale master error: %s", err.Error())
				time.Sleep(30 * time.Second)
				return fmt.Errorf("scale master group: %s", err.Error())
			}
		}

		klog.Infof("[QuorumScale] member center finished")
		return nil
	}
	// 4. do scale and wait
	return ctrl.Result{}, QuorumScale(scale, len(detail.Instances), ms.Spec.Replicas)
}

func QuorumScale(
	mfunc func(cmt int) error,
	current, expect int,
) error {
	if expect == current {
		// no process is needed
		klog.Infof("current ecs count is as expected, nothing to do.")
		return nil
	}
	if expect > current {
		klog.Infof("do scale out: target=%d", expect)
		// scale out should not be controlled.
		return mfunc(expect)
	}
	// max scale in
	max := help.Max((current-1)/2, 1) // max scale num per operate
	klog.Infof("do quorum scale in: target=%d", help.Max(current-max, expect))
	err := mfunc(help.Max(current-max, expect))
	if err != nil {
		return err
	}
	// wait for next retry to scale the remaining node
	// Condition might changed, so we need to force re-enter
	// scale process for safety reason.
	return fmt.Errorf("RetryNextBatch")
}

func findId(
	ecs map[string]provider.Instance, ip string,
) string {
	key := func(ip string) string {
		return fmt.Sprintf("https://%s:2379", ip)
	}
	for _, i := range ecs {
		p := key(i.Ip)
		if ip == p {
			return i.Id
		}
	}
	return ""
}
