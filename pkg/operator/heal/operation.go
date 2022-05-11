package heal

import (
	"context"
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions/etcd"
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
	h "github.com/aoxn/wdrip/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	gerror "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/drain"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

type Manager interface {
	NewOperation(trip *Triple) (Operation, error)
}

func NewOperationMgr(
	prvd pd.Interface,
	recd record.EventRecorder,
	client client.Client,
	drain *drain.Helper,
) *operationMgr {
	return &operationMgr{
		prvd:   prvd,
		recd:   recd,
		client: client,
		drain:  drain,
	}
}

type operationMgr struct {
	drain  *drain.Helper
	prvd   pd.Interface
	recd   record.EventRecorder
	client client.Client
}

func (m *operationMgr) NewOperation(trip *Triple) (Operation, error) {
	return &NodeOperation{trip: trip, manager: m}, nil
}

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
	Cordon(info *NodeInfo, cordon bool) error
	Drain(info *NodeInfo) error
	Restart(info *NodeInfo) error
	Reset(info *NodeInfo) error
	RunCommand(info *NodeInfo, cmd string) error
	LabelNode(info *NodeInfo, lbl map[string]string) error
}

func (m *NodeOperation) Restart(info *NodeInfo) error {
	if info.Instance == nil {
		return fmt.Errorf("empty instance id: %s", info)
	}
	//m.recd.Event(m.node, v1.EventTypeNormal, "NodeHeal.TryRestartECS", m.node.Name)

	klog.Infof("try to restart ecs[%s] to fix node problem", info.Instance.Id)
	err := m.manager.prvd.RestartECS(pd.NewEmptyContext(), info.Instance.Id)
	if err != nil {
		// restart ecs fail, fail immediately
		return errors.Wrapf(err, "restart ecs")
	}
	// todo: fix nodename

	err = h.WaitHeartbeat(m.manager.client, info.GetNodeName(), 3*time.Minute)
	if err != nil {
		return errors.Wrapf(err, "wait node "+
			"ready failed while restarting ecs: %s", info.Instance.Id)
	}
	//m.recd.Event(m.node, v1.EventTypeNormal, "NodeHeal.Succeed", m.node.Name)
	return nil
}

type NodeOperation struct {
	trip    *Triple
	manager *operationMgr
}

func (m *NodeOperation) AdmitECS(info *NodeInfo, duration time.Duration) bool {

	eid := info.Instance
	if eid == nil {
		klog.Warningf("admit ecs empty instance id: %s", info)
		return false
	}
	// figure out whether we can reinitialize node.
	// rule: `wdrip.last.update.time` exists and has been at
	// least N minutes after last re-initialize.
	// tag ecs with `wdrip.last.update.time={now}` when not found.
	for _, t := range eid.Tags {
		if t.Key == WdripLastUpdate {
			klog.Infof("wdrip last re-initialize node at [%s], duration=%s,now=[%s],"+
				" Admitted=%t", t.Val, duration/time.Second, h.Now(), h.After(t.Val.(string), duration))
			return h.After(t.Val.(string), duration)
		}
	}
	klog.Infof("tag %s not found, mark date and return", WdripLastUpdate)
	err := m.manager.prvd.TagECS(
		pd.NewEmptyContext(), eid.Id,
		pd.Value{Key: WdripLastUpdate, Val: h.Now()},
	)
	if err != nil {
		klog.Warningf("canAdmin: mark operation time,%s, %s", eid.Id, err.Error())
	}
	return false
}

func (m *NodeOperation) AdmitNode(info *NodeInfo, duration time.Duration) bool {
	eid := info.Instance
	if eid == nil {
		// empty instance id found, unexpected
		klog.Warningf("admit node, empty instance id found, unexpected: %s", info)
		return false
	}
	node := info.Node
	if node == nil {
		klog.Infof("node object is none, try ecs id: %s", eid.Id)
		nodes, err := h.NodeItems(m.manager.client)
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.Infof("node %s not found, trying to do resetting", eid.Id)
				return true
			}
			klog.Warningf("admin node: %s", err.Error())
			return false
		}
		found := false
		for _, v := range nodes {
			if strings.Contains(
				v.Spec.ProviderID, eid.Id,
			) {
				found = true
				node = &v
				break
			}
		}
		if !found {
			klog.Infof("node %s not found, do resetting", eid.Id)
			return true
		}
	}
	ready, reason := h.NodeHeartbeatReady(node, duration)
	if !ready {
		klog.Infof("node %s not ready, admit node: %s", node.Name, reason)
	}
	return ready
}

func (m *NodeOperation) Reset(info *NodeInfo) error {
	eid := info.Instance
	min := 1 * time.Minute
	if !h.After(eid.CreatedAt, AdmitCreateThrottleTime) {
		klog.Infof("ECS has just been created at %s,"+
			" throttle for %s minutes .............................", eid.CreatedAt, min/time.Minute)
		time.Sleep(min)
		return NewRetry(min)
	}

	// it has been 5 minutes since ecs started.
	// but node still in unknown failed status. try to repair
	if !m.AdmitECS(info, AdmitECSThrottleTime) {
		klog.Infof("[NodeOperation] ecs has been "+
			"processed in less than %d minutes, wait for next retry", AdmitECSThrottleTime/time.Minute)
		return NewRetry(min)
	}

	if !m.AdmitNode(info, AdmitNodeThrottleTime) {
		klog.Infof("node %s %s is not "+
			"allowed to repair. wait for some proper time", eid.Id, eid.Ip)
		return NewRetry(min)
	}

	// todo: drain first
	if err := m.Drain(info); err != nil {
		klog.Warningf("drain node: %s, %s", info.Node.Name, err.Error())
	}
	if info.Role == pd.JoinMasterUserdata {
		err := m.removeEtcd(eid.Ip)
		if err != nil {
			return errors.Wrapf(err, "etcd: %s, %s", eid.Id, eid.Ip)
		}
	}
	spectx := pd.NewContextWithCluster(&m.trip.cluster.Spec)
	// do fix
	data, err := m.manager.prvd.UserData(spectx, info.Role)
	if err != nil {
		return errors.Wrapf(err, "new worker userdata")
	}
	err = m.manager.prvd.ReplaceSystemDisk(spectx, eid.Id, data, pd.Option{})
	if err != nil {

		klog.Warningf("[NodeOperation] replace system disk "+
			"failed: sleep for 15 seconds in case of ddos %s", eid.Id)
		time.Sleep(15 * time.Second)
		return fmt.Errorf("replace system disk failed: %s", err.Error())
	}
	err = m.manager.prvd.TagECS(spectx, eid.Id, pd.Value{Key: WdripLastUpdate, Val: h.Now()})
	if err != nil {
		klog.Warningf("[NodeOperation] replace"+
			"succeed, but tag update time failed: %s", err.Error())
	}
	name := fmt.Sprintf("%s.%s", eid.Ip, eid.Id)
	klog.Infof("[NodeOperation] wait for node becoming ready, %s", name)
	err = h.WaitHeartbeat(m.manager.client, name, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("repair failed, continue on next node, %s", err.Error())
	}
	klog.Infof("[NodeOperation] repair node %s finished.", eid.Id)

	return nil
}

// remove etcd member
func (m *NodeOperation) removeEtcd(ip string) error {
	// remove etcd member first.
	metcd, err := etcd.NewEtcdFromCRD(m.trip.mCRDs, m.trip.cluster, etcd.ETCD_TMP)
	if err != nil {
		return fmt.Errorf("new etcd: %s", err.Error())
	}
	mems, err := metcd.MemberList()
	if err != nil {
		return fmt.Errorf("clean up etcd: member %s", err.Error())
	}
	return metcd.RemoveMember(etcd.FindMemberByIP(mems.Members, ip))
}

func (m *NodeOperation) Drain(info *NodeInfo) error {
	klog.Infof("try cordon node")
	if info.Node == nil {
		klog.Warningf("drain empty node object: %s", info)
		return nil
	}
	err := m.Cordon(info, true)
	if err != nil {
		return fmt.Errorf("cordon node: %s", err.Error())
	}
	pods, errs := m.manager.drain.GetPodsForDeletion(info.Node.Name)
	if errs != nil {
		return gerror.NewAggregate(errs)
	}
	warnings := pods.Warnings()
	if warnings != "" {
		klog.Infof("WARNING: drain %s", warnings)
	}
	err = m.manager.drain.DeleteOrEvictPods(pods.Pods())
	if err != nil {
		pending, newErrs := m.manager.drain.GetPodsForDeletion(info.Node.Name)
		if pending != nil {
			pods := pending.Pods()
			if len(pods) != 0 {
				klog.Infof("there are pending "+
					"pods in node %q when an error occurred: %v", info.Node.Name, err)
				for _, pod := range pods {
					klog.Infof("pods: %s/%s", pod.Namespace, pod.Name)
				}
			}
		}
		if newErrs != nil {
			klog.Infof("drain: GetPodsForDeletion, %s", gerror.NewAggregate(newErrs))
		}
		return err
	}
	klog.Infof("drain %s finished", info.Node.Name)
	return nil
}

func (m *NodeOperation) Cordon(info *NodeInfo, cordon bool) error {
	if info.Node == nil {
		klog.Warningf("cordon[%t] empty node: %s", cordon, info)
		return nil
	}
	help := drain.NewCordonHelper(info.Node)
	help.UpdateIfRequired(cordon)
	err, patcherr := help.PatchOrReplace(m.manager.drain.Client, false)
	if err != nil || patcherr != nil {
		return gerror.NewAggregate([]error{err, patcherr})
	}
	return nil
}

func (m *NodeOperation) RunCommand(info *NodeInfo, cmd string) error {
	eid := info.Instance
	if eid == nil {
		return fmt.Errorf("empty instance information: %s", info)
	}
	ctx := pd.NewContextWithCluster(&m.trip.cluster.Spec)
	_, err := m.manager.prvd.RunCommand(ctx, eid.Id, cmd)
	if err != nil {
		return errors.Wrapf(err, "run command[%s]", cmd)
	}
	return h.WaitHeartbeat(m.manager.client, info.GetNodeName(), 30*time.Second)
}

func (m *NodeOperation) LabelNode(info *NodeInfo, lbl map[string]string) error {
	node := &v1.Node{}
	err := m.manager.client.Get(context.TODO(), client.ObjectKey{Name: info.GetNodeName()}, node)
	if err != nil {
		return errors.Wrapf(err, "[%s] get node", info.GetNodeName())
	}
	diff := func(copy runtime.Object) (client.Object, error) {
		np := copy.(*v1.Node)
		mlabels := np.GetLabels()
		if mlabels == nil {
			mlabels = make(map[string]string)
		}
		for k, v := range lbl {
			mlabels[k] = v
		}
		return np, nil
	}
	klog.Warningf("[%s]label node, %s", info.GetNodeName(), lbl)
	return h.Patch(m.manager.client, node, diff, h.PatchSpec)
}
