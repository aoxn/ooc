package task

import (
	"fmt"
	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

func (r *ReconcileTask) doAutoRepair(task *acv1.Task, node *corev1.Node) (reconcile.Result, error) {

	r.recd.Event(node, corev1.EventTypeNormal, "NodeAutoRepair.TryRestartECS", node.Name)
	klog.Infof("try to restart ecs[%s] to fix node notReady problem", task.Spec.NodeName)
	// TODO: fix this validate
	id := strings.Split(node.Spec.ProviderID, ".")
	err := r.prvd.RestartECS(r.sharedCtx.ProviderCtx(),id[1])
	if err != nil {
		// restart ecs fail, fail immediately
		reason := fmt.Sprintf("restart ecs failed: %s, stop autorepair", err.Error())
		klog.Error(reason)
		r.recd.Event(node, corev1.EventTypeWarning, "NodeAutoRepair.RestartECS.Failed", reason)
		return reconcile.Result{Requeue: false}, PatchStatus(r.client, task, acv1.PhaseFailed, reason)
	}
	err = WaitForNodeReady(r.drain.Client, task, node.Name)
	if err != nil {
		// restart ecs fail, fail immediately
		reason := fmt.Sprintf("wait for node ready failed while restarting ecs: %s, stop autorepair", err.Error())
		klog.Error(reason)
		r.recd.Event(node, corev1.EventTypeWarning, "NodeAutoRepair.WaitNodeReady.Failed", reason)
		return reconcile.Result{Requeue: false}, PatchStatus(r.client, task, acv1.PhaseFailed, reason)
	}
	klog.Infof("kubelet ready, repair complete")
	r.recd.Event(node, corev1.EventTypeNormal, "NodeAutoRepair.Succeed", node.Name)
	return reconcile.Result{}, PatchStatus(r.client, task, acv1.PhaseCompleted, "NodeAutoRepairComplete")
}
