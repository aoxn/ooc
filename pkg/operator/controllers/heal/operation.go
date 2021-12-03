package heal

import (
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func NewRetry(timeout time.Duration) *Retry {
	return &Retry{Duration: timeout}
}

type Retry struct {
	Duration time.Duration
}

func (m *Retry) Error() string {
	return fmt.Sprintf("RetryInNext %d Seconds", m.Duration/time.Second)
}

type Operation interface {
	Execute() error
}

type base struct {
	node   *v1.Node
	prvd   provider.Interface
	recd   record.EventRecorder
	client client.Client
}

func newBase(
	node *v1.Node,
	prvd provider.Interface,
	recd record.EventRecorder,
	client client.Client,
) base {
	return base{
		node:   node,
		prvd:   prvd,
		recd:   recd,
		client: client,
	}
}

func NewRestartECS(
	node *v1.Node,
	prvd provider.Interface,
	recd record.EventRecorder,
	client client.Client,
) *RestartECS {
	return &RestartECS{
		base: newBase(node, prvd, recd, client),
	}
}

type RestartECS struct{ base }

func (m *RestartECS) Execute() error {
	m.recd.Event(m.node, v1.EventTypeNormal, "NodeHeal.TryRestartECS", m.node.Name)
	// TODO: fix this validate
	id := strings.Split(m.node.Spec.ProviderID, ".")
	if len(id) != 2 {
		return fmt.Errorf("ivalid provider id: %s", m.node.Spec.ProviderID)
	}
	klog.Infof("try to restart ecs[%s] to fix node problem", id[1])
	err := m.prvd.RestartECS(provider.NewEmptyContext(), id[1])
	if err != nil {
		// restart ecs fail, fail immediately
		reason := fmt.Sprintf("restart ecs failed: %s, stop noderepair", err.Error())
		klog.Error(reason)
		m.recd.Event(m.node, v1.EventTypeWarning, "NodeHeal.RestartECS.Failed", reason)
		return errors.Wrapf(err, "restart ecs")
	}
	err = h.WaitNodeHeartbeatReady(m.client, m.node.Name)
	if err != nil {
		// restart ecs fail, fail immediately
		reason := fmt.Sprintf("wait node ready failed while restarting ecs: %s", err.Error())
		klog.Error(reason)
		m.recd.Event(m.node, v1.EventTypeWarning, "NodeHeal.WaitNodeHeartbeatReady.Failed", reason)
		return errors.Wrapf(err, reason)
	}
	klog.Infof("kubelet ready, repair complete")
	m.recd.Event(m.node, v1.EventTypeNormal, "NodeHeal.Succeed", m.node.Name)
	return nil
}

func NewResteNode(
	eid  provider.Instance,
	spec *api.ClusterSpec,
	node *v1.Node,
	prvd provider.Interface,
	recd record.EventRecorder,
	client client.Client,
) *ResetNode {
	return &ResetNode{
		eid:  eid, spec: spec,
		base: newBase(node, prvd, recd, client),
	}
}

type ResetNode struct {
	base
	spec *api.ClusterSpec
	eid  provider.Instance
}

func (m *ResetNode) AdmitECS(duration time.Duration) bool {

	// figure out whether we can reinitialize node.
	// rule: `ovm.last.update.time` exists and has been at
	// least N minutes after last re-initialize.
	// tag ecs with `ovm.last.update.time={now}` when not found.
	for _, t := range m.eid.Tags {
		if t.Key == OvmLastUpdate {
			klog.Infof("ovm last re-initialize node at [%s], duration=%s,now=[%s]," +
				" Admitted=%t", t.Val, duration/time.Second, h.Now(), h.After(t.Val.(string), duration))
			return h.After(t.Val.(string), duration)
		}
	}
	klog.Infof("tag %s not found, mark date and return", OvmLastUpdate)
	err := m.prvd.TagECS(
		provider.NewEmptyContext(), m.eid.Id,
		provider.Value{Key: OvmLastUpdate, Val: h.Now()},
	)
	if err != nil {
		klog.Warningf("canAdmin: mark operation time,%s, %s", m.eid.Id, err.Error())
	}
	return false
}

func (m *ResetNode) AdmitNode(duration time.Duration) bool {
	node := m.node
	if node == nil {
		klog.Infof("node object is none, try ecs id: %s", m.eid.Id)
		nodes, err := h.NodeItems(m.client)
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.Infof("node %s not found, trying to do resetting", m.eid.Id)
				return true
			}
			klog.Warningf("admin node: %s", err.Error())
			return false
		}
		found := false
		for _, v := range nodes {
			if strings.Contains(
				v.Spec.ProviderID, m.eid.Id,
			) {
				found = true
				node = &v
				break
			}
		}
		if !found {
			klog.Infof("node %s not found, do resetting", m.eid.Id)
			return true
		}
	}
	ready, reason := h.NodeHeartbeatReady(node, duration)
	if !ready {
		klog.Infof("node %s not ready, admit node: %s", m.node.Name, reason)
	}
	return ready
}

func (m *ResetNode) Execute() error {

	min := 1 * time.Minute
	if !h.After(m.eid.CreatedAt, AdmitCreateThrottleTime) {
		klog.Infof("ECS has just been created at %s,"+
			" throttle for %s minutes .............................", m.eid.CreatedAt, min/time.Minute)
		time.Sleep(min)
		return NewRetry(min)
	}

	// it has been 5 minutes since ecs started.
	// but node still in unknown failed status. try to repair
	if !m.AdmitECS(AdmitECSThrottleTime) {
		klog.Infof("[ResetNode] ecs has been " +
			"processed in less than %d minutes, wait for next retry", AdmitECSThrottleTime/time.Minute)
		return NewRetry(min)
	}

	if !m.AdmitNode(AdmitNodeThrottleTime) {
		klog.Infof("node %s %s is not "+
			"allowed to repair. wait for some proper time", m.eid.Id, m.eid.Ip)
		return NewRetry(min)
	}

	spectx := provider.NewContextWithCluster(m.spec)
	// do fix
	data, err := m.prvd.UserData(spectx, provider.WorkerUserdata)
	if err != nil {
		return errors.Wrapf(err, "new worker userdata")
	}
	err = m.prvd.ReplaceSystemDisk(spectx, m.eid.Id, data, provider.Option{})
	if err != nil {
		// TODO: remove me
		// 	sleep for 15 seconds in case of ddos
		klog.Warningf("[ResetNode] replace system disk "+
			"failed: sleep for 15 seconds in case of ddos %s", m.eid.Id)
		time.Sleep(15 * time.Second)
		return fmt.Errorf("replace system disk failed: %s", err.Error())
	}
	err = m.prvd.TagECS(spectx, m.eid.Id, provider.Value{Key: OvmLastUpdate, Val: h.Now()})
	if err != nil {
		klog.Warningf("[ResetNode] replace"+
			"succeed, but tag update time failed: %s", err.Error())
	}
	name := fmt.Sprintf("%s.%s", m.eid.Ip, m.eid.Id)
	klog.Infof("[ResetNode] wait for node becoming ready, %s", name)
	err = h.WaitNodeHeartbeatReady(m.client, name)
	if err != nil {
		return fmt.Errorf("repair failed, continue on next node, %s", err.Error())
	}
	klog.Infof("[ResetNode] repair node %s finished.", m.eid.Id)

	return nil
}
