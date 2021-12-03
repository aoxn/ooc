package heal

import (
	mctx "context"
	"fmt"
	"github.com/aoxn/ooc/pkg/actions/etcd"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/iaas/provider/dev"
	h "github.com/aoxn/ooc/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
	"sync"
	"time"
)

type MemberHeal struct {
	stack      map[string]provider.Value
	dispatcher chan *Event
	nodes      chan *Event
	state      string
	mutex      sync.RWMutex
	cache 	   cache.Cache
	client     client.Client
	prvd       provider.Interface
}

type Event struct {
	Done chan struct{}
}

const (
	StateClean = "Clean"
	StateDirty = "Dirty"

	OocLastUpdate = "ooc.last.update.time"
)

var _ manager.Runnable = &MemberHeal{}

func NewMemberHeal(
	client client.Client,
	prvd provider.Interface,
) *MemberHeal {
	mem := &MemberHeal{
		client:     client,
		prvd:       prvd,
		dispatcher: make(chan *Event, 0),
		nodes:      make(chan *Event, 0),
	}
	return mem
}

func (m *MemberHeal) Start(ctx mctx.Context) error {
	klog.Info("try start member heal")
	if !m.cache.WaitForCacheSync(ctx){
		return fmt.Errorf("member heal wait for cache sync")
	}
	if m.stack == nil {
		spec,err := h.Cluster(m.client, "kubernetes-cluster")
		if err != nil {
			return errors.Wrap(err, "find my cluster:")
		}
		if spec.Spec.Bind.ResourceId == "" {
			resource, err := m.prvd.GetStackOutPuts(
				provider.NewContext(&spec.Spec),
				&provider.Id{Name: spec.Spec.ClusterID},
			)
			if err != nil {
				return errors.Wrap(err, "provider: list resource")
			}
			spec.Spec.Bind.ResourceId = resource[dev.StackID].Val.(string)
		}
		cctx := provider.NewContext(&spec.Spec)
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

func (m *MemberHeal) InjectCache(cache cache.Cache) error {
	m.cache = cache
	return nil
}

func (m *MemberHeal) InjectClient(me client.Client) error {
	m.client = me
	return nil
}

func (m *MemberHeal) String() string{
	return fmt.Sprintf("member heal: %s", m.state)
}

func (m *MemberHeal) Dirty() bool { return m.state == StateDirty }

// mark mark state of member heal center
func (m *MemberHeal) mark(state string) { m.state = state }

func (m *MemberHeal) NotifyScale(result chan struct{}) { m.dispatcher <- &Event{Done: result} }

func (m *MemberHeal) NotifyNodeEvent(result chan struct{}) { m.nodes <- &Event{Done: result} }

// RemoveFollower remove etcd follower
// only works with exactly 2 etcd members
func (m *MemberHeal) RemoveFollower() (string, error) {
	masters, err := h.Masters(m.client)
	if err != nil {
		return "", fmt.Errorf("member master: %s", err.Error())
	}
	spec, err := h.Cluster(m.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return "", fmt.Errorf("member: spec not found,%s", err.Error())
	}

	metcd, err := etcd.NewEtcdFromMasters(masters, spec, etcd.ETCD_TMP)
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

func (m *MemberHeal) dispatch() {
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
func (m *MemberHeal) DeepCheck() error { return m.doCheck("DeepCheck") }

// ShallowCheck
//   1. it is a cheap check. do not call openapi in this check.
//   2. it is intended to run frequently. every N seconds ?
//   3. it ensures master in ready state and count(MasterCRD)==count(MasterNode)
//   4. try to repair
func (m *MemberHeal) ShallowCheck() error { return m.doCheck("Shallow") }

func (m *MemberHeal) doCheck(checkType string) error {
	mNodes, err := h.Nodes(m.client)
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

		cctx := provider.NewContext(&spec.Spec).WithStack(m.stack)
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
func (m *MemberHeal) Synchronization() error {
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
func (m *MemberHeal) dosync() error {

	spec, err := h.Cluster(m.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return fmt.Errorf("member: spec not found,%s", err.Error())
	}

	mCrds, err := h.Masters(m.client)
	if err != nil {
		return fmt.Errorf("member master: %s", err.Error())
	}

	mNodes, err := h.Nodes(m.client)
	if err != nil {
		return fmt.Errorf("member master: %s", err.Error())
	}

	cctx := provider.NewContext(&spec.Spec).WithStack(m.stack)

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

	detail, err := m.prvd.ScalingGroupDetail(cctx, "", provider.Option{Action: "InstanceIDS"})
	if err != nil {
		return fmt.Errorf("scaling group: %s", err.Error())
	}

	klog.Infof("begin sync: load scaling group detail(len=%d), "+
		"CRD(%d), Nodes(%d) ", len(detail.Instances), len(mCrds), len(mNodes))
	err = CleanUpEtcd(mCrds, detail, spec)
	if err != nil {
		return fmt.Errorf("sync, etcd: %s", err.Error())
	}
	// step 1.
	delCrds, addECS := diff(detail, mCrds)
	if len(delCrds) > 0 {
		// see section 1.1
		// remove extra crds & remove extra etcd member
		err = CleanUpMeta(m, mCrds, delCrds, spec)
		if err != nil {
			return fmt.Errorf("master repair: clean up meta faild, %s", err.Error())
		}
		// continue on next repair action
	}
	klog.Infof("debug master repair diff: "+
		"ShouldBeDeletedCRD=%s, NewlyAddedECS=%s", h.MasterNames(delCrds), h.ECSNames(addECS))
	if len(addECS) > 0 {
		// see section 1.2
		err = m.ResetNode(cctx, addECS, mCrds, spec)
		if err != nil {
			return fmt.Errorf("reinitialize node: %s", err.Error())
		}
	}

	// step 2.
	ecsAdd, nodeDel := diffNodeECS(mNodes, detail)
	klog.Infof("debug master repair diff node: "+
		"ShouldBeDeletedNode=%s, NewlyAddedECS=%s", h.NodeNames(nodeDel), h.ECSNames(ecsAdd))

	if len(nodeDel) > 0 {
		// see section 2.1
		err = CleanUpNode(m, nodeDel)
		if err != nil {
			return fmt.Errorf("master repair: node, %s", err.Error())
		}
	}

	if len(ecsAdd) > 0 {
		klog.Errorf("ecs added or restarted, but node object has not been watched or NotReady.")
		// see section 2.2
		err = m.ResetNode(cctx, ecsAdd, mCrds, spec)
		if err != nil {
			return fmt.Errorf("reinitialize node: %s", err.Error())
		}
	}

	// handle master crd missed situation
	missed := missedMasterCRD(mNodes, mCrds)
	if len(missed) > 0 {
		m.NewMasterCRD(missed)
	}
	return nil
}

const (
	AdmitECSThrottleTime  = 3 * time.Minute
	AdmitNodeThrottleTime = 1 * time.Minute

	AdmitCreateThrottleTime = 5 * time.Minute
)

func (m *MemberHeal) NewMasterCRD(nodes []v1.Node) {
	for _, n := range nodes {
		me := api.Master{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Master",
				APIVersion: api.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: n.Spec.ProviderID,
			},
			Spec: api.MasterSpec{
				Role: "Hybrid",
				ID:   n.Spec.ProviderID,
				IP:   n.Status.Addresses[0].Address,
			},
		}
		err := m.client.Create(mctx.TODO(), &me)
		if err != nil {
			klog.Errorf("create master crd fail: %s", err.Error())
		}
	}
}

func (m *MemberHeal) ResetNode(
	cctx *provider.Context,
	ecs map[string]provider.Instance,
	masters []api.Master, spec *api.Cluster,
) error {

	// remove etcd member
	rmMember := func(ip string) error {
		// remove etcd member first.
		metcd, err := etcd.NewEtcdFromMasters(masters, spec, etcd.ETCD_TMP)
		if err != nil {
			return fmt.Errorf("new etcd: %s", err.Error())
		}
		mems, err := metcd.MemberList()
		if err != nil {
			return fmt.Errorf("clean up etcd: member %s", err.Error())
		}
		return metcd.RemoveMember(etcd.FindMemberByIP(mems.Members, ip))
	}

	// figure out whether we can reinitialize node.
	// rule: `ooc.last.update.time` exists and has been at
	// least N minutes after last re-initialize.
	// tag ecs with `ooc.last.update.time={now}` when not found.
	canAdmitECS := func(
		e provider.Instance, duration time.Duration,
	) bool {
		for _, t := range e.Tags {
			if t.Key == OocLastUpdate {
				klog.Infof("ooc last re-initialize node at %s", t.Val)
				return h.After(t.Val.(string), duration)
			}
		}
		klog.Infof("tag %s not found, mark date and return", OocLastUpdate)
		err := m.prvd.TagECS(cctx, e.Id, provider.Value{Key: OocLastUpdate, Val: h.Now()})
		if err != nil {
			klog.Warningf("canAdmin: mark operation time,%s, %s", e.Id, err.Error())
		}
		return false
	}

	canAdmitNode := func(
		id string, duration time.Duration,
	) bool {
		nodes, err := h.Nodes(m.client)
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.Infof("node %s not found, trying to do resetting", id)
				return true
			}
			klog.Warningf("can admit get nodes: %s", err.Error())
			return false
		}
		for _, v := range nodes {
			if !strings.Contains(v.Spec.ProviderID, id) {
				continue
			}
			cond := h.Condition(v)
			if cond == nil {
				klog.Warningf("condition not found, not expected. do reconcile %s", id)
				return true
			}
			if cond.Status == "True" {
				klog.Infof("node %s in Ready state, do not repair", id)
				return false
			}
			now := cond.LastTransitionTime.Add(duration)
			if now.Before(time.Now()) {
				klog.Infof("node has been UnReady "+
					"for at least %d minutes, can do repair", duration/time.Minute)
				return true
			}
			klog.Infof("node NotReady less than %d minutes, do not repair", duration/time.Minute)
			return false
		}
		klog.Warningf("can admit, node %s not found, do resetting", id)
		return true
	}

	// process each ecs one by one
	for _, e := range ecs {
		if !h.After(e.CreatedAt, AdmitCreateThrottleTime) {
			min := AdmitCreateThrottleTime/time.Minute
			klog.Errorf("[ResetNode] ecs added, but MasterCRD "+
				"has not been watched. wait %d minutes to repair, " +
				"CreateAt=%s",min, e.CreatedAt)
			klog.Infof("WAITING for %s minutes .............................", min)
			// wait for master ready for at least `AdmitCreateThrottleTime` minutes
			// after ecs has been created.
			time.Sleep(AdmitCreateThrottleTime)
			continue
		}
		// it has been 5 minutes since ecs started.
		// but node still in unknown failed status. try to repair
		if !canAdmitECS(e, AdmitECSThrottleTime) {
			klog.Infof("[ResetNode] ecs has been " +
				"processed in less than 3 minutes, wait for next retry")
			continue
		}

		if !canAdmitNode(e.Id, AdmitNodeThrottleTime) {
			klog.Infof("node %s %s is not "+
				"allowed to repair. wait for some proper time", e.Id, e.Ip)
			continue
		}

		klog.Infof("[ResetNode] begain to repair master: %s", e.Id)
		err := rmMember(e.Ip)
		if err != nil {
			return fmt.Errorf("master repair: remove etcd member, %s", err.Error())
		}

		// do fix
		err = m.prvd.ReplaceSystemDisk(cctx, e.Id, provider.Option{})
		if err != nil {
			// TODO: remove me
			// 	sleep for 15 seconds in case of ddos
			klog.Warningf("[ResetNode] replace system disk "+
				"failed: sleep for 15 seconds in case of ddos %s", e.Id)
			time.Sleep(15 * time.Second)
			return fmt.Errorf("replace system disk failed: %s", err.Error())
		}
		err = m.prvd.TagECS(cctx, e.Id, provider.Value{Key: OocLastUpdate, Val: h.Now()})
		if err != nil {
			klog.Warningf("[ResetNode] replace"+
				"succeed, but tag update time failed: %s", err.Error())
		}

		klog.Infof("[ResetNode] wait for master becoming ready, %s, %s", e.Id, e.Ip)
		err = h.WaitNodeReady(m.client, e.Id)
		if err != nil {
			return fmt.Errorf("repair failed, continue on next node, %s", err.Error())
		}
		klog.Infof("[ResetNode] repair master %s finished.", e.Id)

		// return RetryNextSnapshot for next reconcile in case of
		// outdated ECS information.
		// eg. Re-initialize oper might take 2-3 minutes to finish.
		// ECS (or Master or Node) status might have changed until than.
		// so we need to reload env with `RetryNextSnapshot` once the
		// first ECS repair operation is finished.
		return fmt.Errorf("RetryNextSnapshot")
	}
	return nil
}

func CleanUpEtcd(
	all []api.Master,
	ins provider.ScaleGroupDetail,
	spec *api.Cluster,
) error {
	if len(all) == 0 {
		return fmt.Errorf("empty master: retry")
	}
	klog.Infof("trying to ensure etcd member is in correct count[%d]", len(ins.Instances))
	metcd, err := etcd.NewEtcdFromMasters(all, spec, etcd.ETCD_TMP)
	if err != nil {
		return fmt.Errorf("new etcd: %s", err.Error())
	}
	mems, err := metcd.MemberList()
	if err != nil {
		return fmt.Errorf("clean up etcd: member %s", err.Error())
	}

	for _, m := range diffEtcdMember(ins, mems.Members) {
		klog.Infof("ecs[%s] has been removed, clean up corresponding etcd member.", m.IP)
		err = metcd.RemoveMember(m)
		if err != nil {
			return fmt.Errorf("member center: remove member, %s", err.Error())
		}
	}
	return nil
}

func CleanUpMeta(
	m *MemberHeal,
	all []api.Master,
	del []api.Master,
	spec *api.Cluster,
) error {
	// =================================================================
	//
	// count(ecs) < count(master) mean we are in [scaling in]
	for _, d := range del {
		klog.Infof("ecs[%s] has been removed, clean up correspoinding MasterCRD %s", d.Spec.ID, d.Name)
		// 2. clean up master CRDs
		err := m.client.Delete(mctx.TODO(), &d)
		if err != nil {
			return fmt.Errorf("remove master object: %s", err.Error())
		}
	}
	klog.Infof("master clean up succeed")
	return nil
}

func CleanUpNode(
	m *MemberHeal,
	nodes []v1.Node,
) error {
	klog.Infof("try to clean up node meta, %d", len(nodes))
	for _, n := range nodes {
		// corresponding ecs has been deleted. delete node together
		err := m.client.Delete(mctx.TODO(), &n)
		if err != nil {
			return fmt.Errorf("clean up node: %s", err.Error())
		}
		klog.Infof("corresponding ecs has been deleted. delete node together, %s", n.Spec.ProviderID)
	}
	return nil
}

func diff(
	ins provider.ScaleGroupDetail,
	ms []api.Master,
) ([]api.Master, map[string]provider.Instance) {
	id := func(key string) string {
		return strings.Split(key, ".")[1]
	}
	var delCRDs []api.Master
	addECS := make(map[string]provider.Instance)
	for _, m := range ms {
		_, ok := ins.Instances[id(m.Spec.ID)]
		if ok {
			continue
		}
		delCRDs = append(delCRDs, m)
	}
	for _, i := range ins.Instances {
		found := false
		for _, m := range ms {
			if id(m.Spec.ID) == i.Id {
				found = true
				break
			}
		}
		if !found {
			addECS[i.Id] = i
		}
	}
	return delCRDs, addECS
}

func diffEtcdMember(
	ins provider.ScaleGroupDetail,
	mems []etcd.Member,
) []etcd.Member {
	//klog.Infof("debug etcd member list: %s", mems)
	var del []etcd.Member
	for _, m := range mems {
		found := false
		for _, e := range ins.Instances {
			if m.IP == e.Ip {
				found = true
				break
			}
		}
		if !found {
			del = append(del, m)
		}
	}
	if float32(len(del)) > 0.5*float32(len(mems)) {
		klog.Warningf("can not remove major count of "+
			"etcd member: cluster might in an inconsistency state.[members=%d],[remove=%d]", len(mems), len(del))
		return []etcd.Member{}
	}
	return del
}

func diffMasterNode(
	ms []api.Master, nodes []v1.Node,
) ([]api.Master, []v1.Node) {
	var rms []api.Master
	var rnode []v1.Node
	for _, m := range ms {
		found := false
		for _, n := range nodes {
			if m.Name == n.Spec.ProviderID {
				found = true
				break
			}
		}
		if !found {
			rms = append(rms, m)
		}
	}

	for _, n := range nodes {
		found := false
		for _, m := range ms {
			if m.Name == n.Spec.ProviderID {
				found = true
				break
			}
		}
		if !found {
			rnode = append(rnode, n)
		}
	}
	return rms, rnode
}

func diffNodeECS(
	nodes []v1.Node,
	ecs provider.ScaleGroupDetail,
) (map[string]provider.Instance, []v1.Node) {

	id := func(key string) string {
		return strings.Split(key, ".")[1]
	}
	rms := make(map[string]provider.Instance)
	var rnode []v1.Node
	for _, m := range ecs.Instances {
		found := false
		for _, n := range nodes {
			if !h.NodesReady([]v1.Node{n}) {
				continue
			}
			if m.Id == id(n.Spec.ProviderID) {
				found = true
				break
			}
		}
		if !found {
			rms[m.Id] = m
		}
	}

	for _, n := range nodes {
		if !h.NodesReady([]v1.Node{n}) {
			klog.Infof("diff, warning node not ready: %s", n.Name)
		}
		_, found := ecs.Instances[id(n.Spec.ProviderID)]
		if !found {
			rnode = append(rnode, n)
		}
	}
	return rms, rnode
}

func missedMasterCRD(
	nodes []v1.Node,
	master []api.Master,
) []v1.Node {
	var result []v1.Node
	for _, n := range nodes {
		found := false
		for _, m := range master {
			if n.Spec.ProviderID == m.Name {
				found = true
				break
			}
		}
		if !found {
			result = append(result, n)
		}
	}
	return result
}

