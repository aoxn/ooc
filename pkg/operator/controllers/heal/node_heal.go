package heal

import (
	mctx "context"
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/iaas/provider/alibaba"
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sync"
	"time"
)

type NodeHeal struct {
	stack      map[string]provider.Value
	dispatcher chan *Event
	nodes      chan *Event
	state      string
	mutex      sync.RWMutex
	cache      cache.Cache
	client     client.Client
	prvd       provider.Interface
}

var _ manager.Runnable = &NodeHeal{}

func NewNodeHeal(
	client client.Client,
	prvd provider.Interface,
) *NodeHeal {
	mem := &NodeHeal{
		client:     client,
		prvd:       prvd,
		dispatcher: make(chan *Event, 0),
		nodes:      make(chan *Event, 0),
	}
	return mem
}

func (m *NodeHeal) Start(ctx mctx.Context) error {
	klog.Info("try start member heal")
	if !m.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("member heal wait for cache sync")
	}
	if m.stack == nil {
		spec, err := h.Cluster(m.client, "kubernetes-cluster")
		if err != nil {
			return errors.Wrap(err, "find my cluster:")
		}
		if spec.Spec.Bind.ResourceId == "" {
			resource, err := m.prvd.GetStackOutPuts(
				provider.NewContextWithCluster(&spec.Spec),
				&api.ClusterId{ObjectMeta: metav1.ObjectMeta{Name: spec.Spec.ClusterID}},
			)
			if err != nil {
				return errors.Wrap(err, "provider: list resource")
			}
			spec.Spec.Bind.ResourceId = resource[alibaba.StackID].Val.(string)
		}
		cctx := provider.NewContextWithCluster(&spec.Spec)
		m.stack, err = h.LoadStack(m.prvd, cctx, spec)
		if err != nil {
			return errors.Wrap(err, "member center: lazy load stack")
		}
	}
	if h.InCluster() {

		klog.Infof("sync once on member heal center start up")
		err := m.Synchronization()
		if err != nil {
			klog.Errorf("Synchronization master state failed. %s", err.Error())
		}
		// run in single goroutine
		wait.Forever(m.dispatch, 5*time.Second)
	}
	klog.Infof("[skip] master member heal in debug mod")
	return nil
}

func (m *NodeHeal) InjectCache(cache cache.Cache) error {
	m.cache = cache
	return nil
}

func (m *NodeHeal) InjectClient(me client.Client) error {
	m.client = me
	return nil
}

func (m *NodeHeal) String() string {
	return fmt.Sprintf("member heal: %s", m.state)
}

func (m *NodeHeal) Dirty() bool { return m.state == StateDirty }

// mark mark state of member heal center
func (m *NodeHeal) mark(state string) { m.state = state }

func (m *NodeHeal) NotifyScale(result chan struct{}) { m.dispatcher <- &Event{Done: result} }

func (m *NodeHeal) NotifyNodeEvent(result chan struct{}) { m.nodes <- &Event{Done: result} }


func (m *NodeHeal) dispatch() {
	tick := time.NewTicker(1 * time.Minute)
	for {
		select {
		case event := <-m.dispatcher:
			klog.Infof("deep check: might scaling event: %v", event)
			err := m.DeepCheck()
			if err != nil {
				klog.Errorf("try to sync master state, but failed: %s", err.Error())
			}
			if event.Done != nil {
				event.Done <- struct{}{}
				close(event.Done)
			}
		case event := <-m.nodes:
			klog.Infof("routine check: node update")
			err := m.ShallowCheck()
			if err != nil {
				klog.Errorf("try to ensure master state on node change: %s", err.Error())
			}
			if event.Done != nil {
				event.Done <- struct{}{}
				close(event.Done)
			}
		case <-tick.C:
			klog.Infof("routine check: every minute member steady check")
			err := m.ShallowCheck()
			if err != nil {
				klog.Errorf("minute steady check: %s", err.Error())
			}
		}
	}
}

// DeepCheck
//   1. run shallow check
//   2. call ess openapi to check ecs status.
//   3. try to reapir
func (m *NodeHeal) DeepCheck() error { return m.doCheck("DeepCheck") }

// ShallowCheck
//   1. it is a cheap check. do not call openapi in this check.
//   2. it is intended to run frequently. every N seconds ?
//   3. it ensures master in ready state and count(MasterCRD)==count(MasterNode)
//   4. try to repair
func (m *NodeHeal) ShallowCheck() error { return m.doCheck("Shallow") }

func (m *NodeHeal) doCheck(checkType string) error {
	mNodes, err := h.MasterNodes(m.client)
	if err != nil {
		return fmt.Errorf("member master: %s", err.Error())
	}
	if !h.NodesReady(mNodes) {
		klog.Infof("some nodes not ready, trigger sync")
		// do repair
		return m.Synchronization()
	}
	mCrd, err := h.Masters(m.client)
	if err != nil {
		return fmt.Errorf("member master: %s", err.Error())
	}
	if len(mCrd) != len(mNodes) {
		klog.Infof("MasterCRD(%d) != MasterNode(%d), trigger sync", len(mCrd), len(mNodes))
		// do repair
		return m.Synchronization()
	}
	rms, rnode := diffMasterNode(mCrd, mNodes)
	if len(rms) != 0 || len(rnode) != 0 {
		klog.Infof("remove MasterCRD(%d) or MasterNodes(%d) found, trigger sync", len(rms), len(rnode))
		// do repair
		return m.Synchronization()
	}
	// run deep check on some important event. eg. scaling out & in
	if checkType == "DeepCheck" {
		klog.Infof("run deep check")
		spec, err := h.Cluster(m.client, api.KUBERNETES_CLUSTER)
		if err != nil {
			return fmt.Errorf("member: spec not found,%s", err.Error())
		}

		cctx := provider.NewContextWithCluster(&spec.Spec).WithStack(m.stack)
		detail, err := m.prvd.ScalingGroupDetail(cctx, "", provider.Option{Action: "InstanceIDS"})
		if err != nil {
			return fmt.Errorf("scaling group: %s", err.Error())
		}

		// step 1.
		delCrds, addECS := diff(detail, mCrd)
		if len(delCrds) > 0 || len(addECS) > 0 {
			klog.Infof("compare(ScalingGroup, MasterCRD): "+
				"MasterCRD(remove=%d) or ECS(add=%d) found, trigger sync", len(delCrds), len(addECS))
			return m.Synchronization()
		}
	}
	return nil
}

// Synchronization
// sync master state, clean up etcd member if master ecs has been removed
// could be execute every 5 minutes.
func (m *NodeHeal) Synchronization() error {
	// protected by lock.
	// Attention: must not be executed concurrently.
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return wait.PollImmediate(
		5*time.Second,
		9*time.Minute,
		func() (done bool, err error) {
			// mark state dirty before doing repair work
			m.mark(StateDirty)
			klog.Infof("start synchronization: mark %s", StateDirty)
			err = m.dosync()
			if err != nil {
				klog.Infof("Synchronization master state: %s", err.Error())
				return false, nil
			}
			m.mark(StateClean)
			klog.Infof("master repair: finished, mark state[%s]", StateClean)
			return true, nil
		},
	)
}

// dosync
// do not call dosync outside of Synchronization
func (m *NodeHeal) dosync() error {

	return nil
}
