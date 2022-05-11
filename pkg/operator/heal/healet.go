package heal

import (
	mctx "context"
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions/etcd"
	api "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
	h "github.com/aoxn/wdrip/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	app "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/drain"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"sync"
	"time"
)

type Healet struct {
	tripGetter TripleGetter
	operation  Manager
	nodes      chan *Event
	nodepool   chan *Event
	infra      Infra
	state      string
	mutex      sync.RWMutex
	cache      cache.Cache
	client     client.Client
	initSpec   *api.Cluster

	//prvd       pd.Interface
	isCtrlPlanInChecking bool
}

type Event struct {
	Object runtime.Object
	Done   chan struct{}
}

const (
	StateClean = "Clean"
	StateDirty = "Healthy"

	WdripLastUpdate = "wdrip.last.update.time"
)

var _ manager.Runnable = &Healet{}

func NewHealet(
	spec *api.Cluster,
	client client.Client,
	prvd pd.Interface,
	drain *drain.Helper,
) (*Healet, error) {
	var err error
	if spec == nil {
		spec, err = h.Cluster(client, api.KUBERNETES_CLUSTER)
		if err != nil {
			return nil, errors.Wrapf(err, "find cluster spec")
		}
	}

	infra, err := NewInfraManager(spec, prvd)
	if err != nil {
		return nil, errors.Wrapf(err, "new infra manager")
	}

	mem := &Healet{
		initSpec:   spec,
		tripGetter: NewTripleGetter(infra, client),
		client:     client,
		infra:      infra,
		operation:  NewOperationMgr(prvd, nil, client, drain),
		nodes:      make(chan *Event, 0),
		nodepool:   make(chan *Event, 0),
	}
	return mem, nil
}

func (m *Healet) Start(ctx mctx.Context) error {
	klog.Info("try start member heal")
	if !m.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("member heal wait for cache sync")
	}

	InitializeDefaultCR(m.client)

	//time.Sleep(10000 * time.Minute)
	if h.InCluster() {
		silent := m.initSpec.Spec.SilentTime
		if silent == 0 {
			silent = 60
		}
		klog.Infof("silent heal time on startup is %ds", silent)
		time.Sleep(time.Duration(silent) * time.Second)
		klog.Infof("sync once on member heal center start up")
		err := m.FixMasterNode()
		if err != nil {
			klog.Errorf("Synchronization master state failed. %s", err.Error())
		}
		// run in single goroutine
		wait.Forever(m.run, 5*time.Second)
	}
	klog.Infof("[skip] master member heal in debug mod")
	return nil
}

func (m *Healet) InjectCache(cache cache.Cache) error {
	m.cache = cache
	return nil
}

func (m *Healet) InjectClient(me client.Client) error {
	m.client = me
	return nil
}

func (m *Healet) String() string {
	return fmt.Sprintf("member heal: %s", m.state)
}

func (m *Healet) Healthy() bool {
	trip, err := NewTripleMaster(m.tripGetter)
	if err != nil {
		klog.Errorf("healthy condition: %s", err.Error())
		return false
	}
	return trip.SystemInConsistency()
}

// mark mark state of member heal center
func (m *Healet) mark(state string) { m.state = state }

func (m *Healet) NotifyNodeEvent(result chan struct{}) { m.nodes <- &Event{Done: result} }

func (m *Healet) NotifyNodePoolEvent(result chan struct{}) { m.nodes <- &Event{Done: result} }

// InitializeDefaultCR initialize default nodepool and masterset
func InitializeDefaultCR(cfg client.Client) {

	mset := &api.MasterSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.MasterSetKind,
			APIVersion: api.SchemeGroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "masterset",
			Namespace: "kube-system",
		},
		Spec: api.MasterSetSpec{Replicas: 1},
	}
	err := cfg.Get(mctx.TODO(), client.ObjectKey{Name: mset.Name}, mset)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = cfg.Create(mctx.TODO(), mset)
			if err != nil {
				klog.Warningf("ensure masterset failed: %s", err.Error())
			}
		} else {
			klog.Infof("ensure masterset, find masterset: %s", err.Error())
		}
	}

	npname := "default-nodepool"
	dnp := &api.NodePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.NodePoolKind,
			APIVersion: api.SchemeGroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      npname,
			Namespace: "kube-system",
		},
		Spec: api.NodePoolSpec{
			NodePoolID: npname,
			Infra: api.Infra{
				CPU: 4, Mem: 8, DesiredCapacity: 0,
				ImageId: "centos_7_9_x64_20G_alibase_20210623.vhd",
			},
		},
	}
	err = cfg.Get(mctx.TODO(), client.ObjectKey{Name: dnp.Name}, dnp)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = cfg.Create(mctx.TODO(), dnp)
			if err != nil {
				klog.Warningf("ensure default nodepool failed: %s", err.Error())
			}
		} else {
			klog.Infof("ensure default nodepool, find default nodepool: %s", err.Error())
		}
	}
}

// RemoveFollower remove etcd follower
// only works with exactly 2 etcd members
func (m *Healet) RemoveFollower() (string, error) {
	masters, err := h.MasterCRDS(m.client)
	if err != nil {
		return "", fmt.Errorf("member master: %s", err.Error())
	}
	spec, err := h.Cluster(m.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return "", fmt.Errorf("member: spec not found,%s", err.Error())
	}

	metcd, err := etcd.NewEtcdFromCRD(masters, spec, etcd.ETCD_TMP)
	if err != nil {
		return "", fmt.Errorf("new etcd: %s", err.Error())
	}
	endps, err := metcd.Endpoints()
	if err != nil {
		return "", fmt.Errorf("endpoint status fail: %s", err.Error())
	}

	if len(endps) != 2 {
		return "", fmt.Errorf("remove follower can only work with 2 etcd member")
	}

	if len(endps) == 1 {
		klog.Infof("only one etcd replica remain, skip remove operation")
		return "", nil
	}
	for _, end := range endps {
		klog.Infof("etcd endpoint: %+v", end)
		leader := end.Status.Leader
		memid := end.Status.Header.MemberID
		if leader.Cmp(memid) == 0 {
			continue
		}
		mem := etcd.Member{ID: end.Status.Header.MemberID}
		err = metcd.RemoveMember(mem)
		if err != nil {
			return end.Endpoint, fmt.Errorf("remove member: %x, %s", mem.ID, err.Error())
		}
		return end.Endpoint, nil
	}
	return "", fmt.Errorf("empty endpoint: %v", endps)
}

// run deprecated
func (m *Healet) run() {

	tick := time.NewTicker(60 * time.Second)
	for {
		select {
		case _ = <-m.nodes:
			klog.Infof("routine check: node update, do nothing")
		case event := <-m.nodepool:
			np, ok := event.Object.(*api.NodePool)
			if !ok {
				klog.Warningf("not nodepool object, skip")
				return
			}

			err := m.FixNodePool(np)
			if err != nil {
				klog.Errorf("nodepool fix check: %s", err.Error())
			}
			klog.Infof("[%s]routine node pool fix finished", np.Name)
		case <-tick.C:
			err := m.FixMasterNode()
			if err != nil {
				klog.Errorf("master steady check: %s", err.Error())
			}
			klog.Infof("[master.fix]routine check: 60 seconds steady tick finished")
		}
	}
}

func (m *Healet) FixNodePool(pool *api.NodePool) error {
	if m.isCtrlPlanInChecking {
		return fmt.Errorf("control plane is in checking, "+
			"wait for next tick to fix node pool: %s", pool.Name)
	}
	// fix labels
	detail, err := m.tripGetter.GetNodePoolECS(pool)
	if err != nil {
		//if strings.Contains(err.Error(), "InvalidVPC") {
		//	// Fix InvalidVPC error
		//	diff := func(copy runtime.Object) (client.Object, error) {
		//		np := copy.(*api.NodePool)
		//		np.Spec.Infra.Bind = nil
		//		pool.Spec.Infra.Bind = nil
		//		return np, nil
		//	}
		//	klog.Warningf("clear nodepool bind infra "+
		//		"for invalid vpc: [%s], [%s]", pool.Name, pool.Spec.Infra.Bind)
		//	return h.Patch(m.client, pool, diff, h.PatchSpec)
		//}
		return errors.Wrap(err, "get nodepool ecs")
	}
	nodes, err := h.NodeItems(m.client)
	if err != nil {
		return errors.Wrapf(err, "get all nodes")
	}

	klog.Infof("debug node list: total %d nodes", len(nodes))
	// names has no nodepool labels
	var names []v1.Node
	for _, d := range detail {
		for _, n := range nodes {
			if strings.Contains(n.Spec.ProviderID, d.Id) {
				if !h.HasNodePoolID(n, pool.Name) {
					names = append(names, n)
				}
				break
			}
		}
	}

	klog.Infof("[%s] %d node has no nodepool labels", pool.Name, len(names))
	for _, n := range names {
		diff := func(copy runtime.Object) (client.Object, error) {
			node := copy.(*v1.Node)
			if node.Labels == nil {
				node.Labels = map[string]string{}
			}
			node.Labels["np.wdrip.io/id"] = pool.Name
			return node, nil
		}
		klog.Warningf("patch nodepool label [np.wdrip.io/id=%s] for %s", pool.Name, n.Name)
		err := h.Patch(m.client, &n, diff, h.PatchSpec)
		if err != nil {
			return errors.Wrapf(err, "patch nodepool labels")
		}
	}

	trip, err := NewTripleWorker(m.tripGetter, pool)
	if err != nil {
		return errors.Wrap(err, "get Triple")
	}
	klog.Infof("[step 1] trying to ensure nodepool %s, %s", pool.Name, trip)
	return m.FixUpNode(trip)
}

// FixMasterNode
// trying to fix node meta, node status
func (m *Healet) FixMasterNode() error {
	// prevent from concurrent fix.
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.isCtrlPlanInChecking = true
	defer func() { m.isCtrlPlanInChecking = false }()

	trip, err := NewTripleMaster(m.tripGetter)
	if err != nil {
		return errors.Wrap(err, "get Triple")
	}

	// repair sequence.
	// attention: clean up metadata first. MasterCRD & Etcd Member.
	// 1. diff check count(MasterECS)==count(MasterCRD), ensure metadata, etcd etc.
	//      1). case count(MasterECS) < count(MasterCRD). remove extra
	//     		MasterCRD, and remove extra etcd member
	//      2). case count(MasterECS) > count(MasterCRD). some ecs
	//     		failed(or has not yet) to join in. clean up etcd member.
	//    		and replace systemdisk, and make a re-join process.
	// 			control repair rate. time is critical.
	// 2. diff check count(MasterECS)==count(MasterNode), ensure node meta, label etc.
	// 		1). case count(MasterECS) < count(MasterNode). some ECS has
	//			been removed. Just do delete node metadata.
	// 		2). case count(MasterECS) > count(MasterNode). some ECS has
	//			failed(or has not yet) to join in. same as section 1.2
	// 3. finished

	// step 1.
	klog.Infof("[step 1] sync metadata: %s", trip)
	// see section 1.1
	// remove extra crds & remove extra etcd member
	err = m.FixUpMeta(trip)
	if err != nil {
		return fmt.Errorf("master repair: clean up meta faild, %s", err.Error())
	}

	klog.Infof("[step 2] sync etcd: %s", trip)
	err = m.FixUpEtcd(trip)
	if err != nil {
		return fmt.Errorf("sync, etcd: %s", err.Error())
	}

	// continue on next repair action
	klog.Infof("[step 3] sync master node: %s", trip)
	// see section 1.2
	return m.FixUpNode(trip)
}

const (
	AdmitECSThrottleTime  = 3 * time.Minute
	AdmitNodeThrottleTime = 1 * time.Minute

	AdmitCreateThrottleTime = 5 * time.Minute
)

func (m *Healet) FixMasterCRD(nodes []NodeInfo) error {
	for _, n := range nodes {
		me := api.Master{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Master",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: n.Node.Spec.ProviderID,
			},
			Spec: api.MasterSpec{
				Role: "Hybrid",
				ID:   n.Node.Spec.ProviderID,
				IP:   n.Node.Status.Addresses[0].Address,
			},
		}
		err := m.client.Create(mctx.TODO(), &me)
		if err != nil {
			klog.Errorf("create master crd fail: %s", err.Error())
		}
	}
	return nil
}

func (m *Healet) FixUpNode(trip *Triple) error {
	addition, deletion := trip.InstanceNodeDiff()
	klog.Infof("try to clean up node meta, addition=%d, deletion=%d", len(addition), len(deletion))
	for _, n := range deletion {
		// corresponding ecs has been deleted. delete node together
		err := m.client.Delete(mctx.TODO(), n.Node)
		if err != nil {
			return fmt.Errorf("clean up node: %s", err.Error())
		}

		klog.Infof("corresponding ecs has been "+
			"deleted. delete node together, %s", n.Node.Spec.ProviderID)
	}

	nop, err := m.operation.NewOperation(trip)
	if err != nil {
		return errors.Wrapf(err, "new node operation: %s", trip)
	}

	if len(addition) <= 0 {
		info := trip.UnReadyNodeList()
		if len(info) <= 0 {
			// nothing to fix
			return nil
		}
		klog.Infof("unready nodes %d, trying to fix", len(info))
		// todo: fix master first
		// fix not ready node one by one randomly. wait for next tick.
		return fixUpHard(nop, &info[randn(len(info))])
	}
	noinfo := addition[0]
	klog.Infof("new ecs %s without ready node object, trying to fix", noinfo.GetNodeName())
	// fix one by one
	return fixUpHard(nop, &noinfo)
}

// fixUpSoft deprecated
func fixUpSoft(nop Operation, info *NodeInfo) error {
	klog.Infof("[%s]trying to fix unready node", info.Instance.Id)
	klog.Infof("[%s]trying to restart kubelet", info.Instance.Id)
	// 1. runcommand restart kubelet & wait
	err := nop.RunCommand(info, "systemctl restart kubelet")
	if err == nil {
		klog.Infof("[%s]fixed node with [RestartKubelet]", info.Instance.Id)
		return nil
	}
	klog.Warningf("[%s]failed to fix[NotFixed] with [RestartKubelet], %s", info.Instance.Id, err.Error())
	// todo: node without master label
	klog.Infof("[%s]try restart ECS           ", info.Instance.Id)

	// 2. restart ecs & wait
	err = nop.Restart(info)
	if err == nil {
		klog.Infof("[%s]fixed node with [RestartECS]", info.Instance.Id)
		return nil
	}
	klog.Warningf("[%s]failed to fix[NotFixed] with [RestartECS], %s", info.Instance.Id, err.Error())
	klog.Infof("[%s]trying to reset ECS", info.Instance.Id)
	//3. reset node & wait
	err = nop.Reset(info)
	if err == nil {
		klog.Infof("[%s]fixed node with [ResetECS]", info.Instance.Id)
		return nil
	}
	klog.Infof("[%s]failed to fix[NotFixed] with [ResetECS], %s", info.Instance.Id, err.Error())
	return nil
}

func fixUpHard(nop Operation, info *NodeInfo) error {
	klog.Infof("[%s]trying to restart kubelet", info.Instance.Id)
	// 1. runcommand restart kubelet & wait
	err := nop.RunCommand(info, "systemctl restart kubelet")
	if err == nil {
		klog.Infof("[%s]fixed node with [RestartKubelet]", info.Instance.Id)
		if info.Role != pd.JoinMasterUserdata {
			return nil
		}
		klog.Infof("[%s]mark controlplane labels: master,control-plane", info.Instance.Id)
		lbl := map[string]string{
			"node-role.kubernetes.io/master":        "",
			"node-role.kubernetes.io/control-plane": "",
		}
		return nop.LabelNode(info, lbl)
	}
	klog.Warningf("[%s]failed to fix[NotFixed] with [RestartKubelet], %s", info.Instance.Id, err.Error())
	klog.Infof("[%s]trying to fix unready node with reset", info.Instance.Id)
	err = nop.Reset(info)
	if err == nil {
		klog.Infof("[%s]fixed node with [ResetECS]", info.Instance.Id)
		return nil
	}
	klog.Infof("[%s]failed to fix[NotFixed] with [ResetECS], %s", info.Instance.Id, err.Error())
	return nil
}

func (m *Healet) FixUpEtcd(trip *Triple) error {
	if len(trip.mCRDs) == 0 {
		return fmt.Errorf("empty master: abort etcd fix")
	}
	klog.Infof("trying to ensure etcd is in expected count[%d]", len(trip.instances))
	metcd, err := etcd.NewEtcdFromCRD(trip.mCRDs, trip.cluster, etcd.ETCD_TMP)
	if err != nil {
		return fmt.Errorf("new etcd: %s", err.Error())
	}
	mems, err := metcd.MemberList()
	if err != nil {
		return fmt.Errorf("clean up etcd: member %s", err.Error())
	}

	for _, m := range trip.EtcdMemDiff(mems.Members) {
		klog.Infof("ecs[%s] has been removed, clean up corresponding etcd member.", m.IP)
		err = metcd.RemoveMember(m)
		if err != nil {
			return fmt.Errorf("member center: remove member, %s", err.Error())
		}
	}
	return nil
}

func (m *Healet) FixUpMeta(trip *Triple) error {
	// handle master crd missed situation
	deletions, missed := trip.MasterCRDDiff()

	for _, d := range deletions {
		klog.Infof("ecs has been removed, "+
			"clean up corresponding MasterCRD %s", d.Resource.Name)
		// 2. clean up master CRDs
		err := m.client.Delete(mctx.TODO(), d.Resource)
		if err != nil {
			return fmt.Errorf("remove master object: %s", err.Error())
		}
	}
	klog.Infof("master clean up succeed")

	return m.FixMasterCRD(missed)
}

func (m *Healet) FixWDRIP() error {
	//this function is for wdrip monitor.
	//There are failing cases that master node object has gone
	//unexpectedly but controlplane still ready with wdrip failed
	// to deploy on masters, because no available master nodes.
	// The monitor programe is indicated for this case.

	wdrip := &app.Deployment{}
	jkey := client.ObjectKey{Namespace: "kube-system", Name: "wdrip"}
	err := m.client.Get(mctx.TODO(), jkey, wdrip)
	if err != nil {
		klog.Warningf("failed to query wdrip status: %s", err.Error())
		return nil
	}

	if wdrip.Status.ReadyReplicas != 0 {
		// wdrip replicas is good
		klog.Infof("[FixWDRIP]desired wdrip count[%d] is great than 0, skip", wdrip.Status.ReadyReplicas)
		return nil
	}
	// that means no master to schedule, trying to fix

	trip, err := NewTripleMaster(m.tripGetter)
	if err != nil {
		return errors.Wrap(err, "get Triple")
	}
	err = m.FixUpMeta(trip)
	if err != nil {
		return errors.Wrapf(err, "fix wdrip and meta")
	}
	fixup := func(info *NodeInfo) error {
		klog.Infof("trying to fix wdrip: %s", info)
		nop, err := m.operation.NewOperation(trip)
		if err != nil {
			return errors.Wrapf(err, "[FixWDRIP] new operations")
		}
		err = nop.RunCommand(info, "systemctl restart kubelet")
		if err != nil {
			klog.Warningf("[FixWDRIP] restart kubelet: %s", err.Error())
		}
		lbl := map[string]string{
			"node-role.kubernetes.io/master":        "",
			"node-role.kubernetes.io/control-plane": "",
		}
		return nop.LabelNode(info, lbl)
	}
	additions, _ := trip.InstanceNodeDiff()
	if len(additions) > 0 {
		return fixup(&additions[randn(len(additions))])
	}
	return nil
}

func randn(n int) int {
	if n <= 1 {
		return 0
	}
	return rand.Intn(n - 1)
}

type NodeInfo struct {
	// Role etc. provider.WorkerUserdata
	Role     string
	Instance *pd.Instance
	Node     *v1.Node
	Resource *api.Master
	GetName  func() string
}

func (n NodeInfo) String() string {
	var id []string
	if n.Instance != nil {
		id = append(id, fmt.Sprintf("instance=%s", n.Instance.Ip))
	}
	if n.Node != nil {
		id = append(id, fmt.Sprintf("node=%s", n.Node.Name))
	}
	if n.Resource != nil {
		id = append(id, fmt.Sprintf("mcrd=%s", n.Resource.Name))
	}
	id = append(id, n.Role)
	return fmt.Sprintf("[NodeINFO](%s)", strings.Join(id, ","))
}

func (n *NodeInfo) GetNodeName() string {
	if n.Node != nil {
		return n.Node.Name
	}
	if n.GetName != nil {
		return n.GetName()
	}
	// todo: fix node name
	// fallback to default name pattern, this might not be suitable
	return fmt.Sprintf("%s.%s", n.Instance.Ip, n.Instance.Id)
}

func ToList(
	i map[string]pd.Instance,
) []pd.Instance {

	var add []pd.Instance
	for _, v := range i {
		add = append(add, v)
	}
	return add
}
