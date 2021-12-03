package help

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func WaitNodeHeartbeatReady(
	client client.Client,
	nodeName string,
) error {
	klog.Infof("wait for node ready")
	waitReady := func() (done bool, err error) {
		node, err := Node(client,nodeName)
		if err != nil {
			klog.Infof("wait node ready: %s", err.Error())
			return false, nil
		}
		ready, reason := nodeHeartbeatReady(node, 0)
		if !ready {
			klog.Warningf("kubelet not ready yet: %s %s", nodeName, reason)
			return false, nil
		}
		klog.Infof("kubelet ready: %s", nodeName)
		return true, nil
	}
	return wait.PollImmediate(
		10*time.Second, 3*time.Minute,
		func() (done bool, err error) {
			// wait for node status steady in 6s
			for i := 0; i < 3; i++ {
				ready, err := waitReady()
				if err != nil || !ready {
					return ready, err
				}
				klog.Infof("node[%s] status ready, wait for status steady[%d]", nodeName, i)
				time.Sleep(2 * time.Second)
			}
			return true, nil
		},
	)
}

// NodeHeartbeatReady timeout seconds
func NodeHeartbeatReady(node *v1.Node,timeout time.Duration) (bool , string) {return nodeHeartbeatReady(node, timeout)}

func nodeHeartbeatReady(node *v1.Node, timeout time.Duration) (bool, string) {
	cond := findCondition(node.Status.Conditions, v1.NodeReady)
	if cond.Type != v1.NodeReady {
		klog.Infof("ready condition type not found,%s", node.Name)
		return true, "ConditionNotFound"
	}
	if cond.Status == v1.ConditionFalse ||
		cond.Status == v1.ConditionUnknown {
		klog.Infof("node %s in not ready " +
			"state[%s], wait heartbeat timeout: %s",node.Name, cond.Status, cond.LastHeartbeatTime)
	}
	return cond.Status == v1.ConditionTrue && isHeartbeatNormal(cond,timeout), cond.Reason
}

const HeartBeatTimeOut = 2 * time.Minute

func isHeartbeatNormal(cond v1.NodeCondition, timeout time.Duration) bool {
	duration := HeartBeatTimeOut
	if timeout != 0 {
		duration = timeout
	}

	// todo:
	//     fix time.Zone problem.
	ago := metav1.NewTime(time.Now().Add(-1 * duration))
	// heartbeat hasn`t been updated for at least 2 minute
	return !cond.LastHeartbeatTime.Before(&ago)
}

func findCondition(
	conds []v1.NodeCondition,
	typ v1.NodeConditionType,
) v1.NodeCondition {
	for i := range conds {
		if conds[i].Type == typ {
			return conds[i]
		}
	}
	klog.Infof("condition type %s not found,skip", typ)
	return v1.NodeCondition{}
}



func NodeReady(n *v1.Node) bool {
	return NodesReady([]v1.Node{*n})
}

func NodesReady(nodes []v1.Node) bool {
	for _, n := range nodes {
		condi := n.Status.Conditions
		if condi == nil {
			continue
		}
		for _, con := range condi {
			if con.Type != "Ready" {
				// only kubelet ready status is take cared
				continue
			}
			if con.Status != "True" {
				// short cut search
				return false
			}
		}
	}
	// default to true.
	return true
}
