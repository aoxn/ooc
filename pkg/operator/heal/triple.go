package heal

import (
	"fmt"
	"github.com/aoxn/ovm/pkg/actions/etcd"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	pd "github.com/aoxn/ovm/pkg/iaas/provider"
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type TripleGetter interface {
	GetClusterItem() (*api.Cluster, error)
	GetMasterNodeList() ([]v1.Node, error)
	GetWorkerNodeList(np *api.NodePool) ([]v1.Node, error)
	GetMasterCR() ([]api.Master, error)
	GetControlPlaneECS() (map[string]pd.Instance, error)
	GetNodePoolECS(np *api.NodePool) (map[string]pd.Instance, error)
}

func NewTripleGetter(infra Infra, m client.Client) *wTripleGetter {
	return &wTripleGetter{infra: infra, client: m}
}

type wTripleGetter struct {
	infra  Infra
	client client.Client
}

func (w *wTripleGetter) GetClusterItem() (*api.Cluster, error) {
	return h.Cluster(w.client, api.KUBERNETES_CLUSTER)
}

func (w *wTripleGetter) GetMasterNodeList() ([]v1.Node, error) { return h.MasterNodes(w.client) }

func (w *wTripleGetter) GetWorkerNodeList(np *api.NodePool) ([]v1.Node, error) {
	return h.NodePoolItems(w.client, np)
}

func (w *wTripleGetter) GetMasterCR() ([]api.Master, error) { return h.MasterCRDS(w.client) }

func (w *wTripleGetter) GetControlPlaneECS() (map[string]pd.Instance, error) {
	return w.infra.ControlPlaneECS()
}

func (w *wTripleGetter) GetNodePoolECS(np *api.NodePool) (map[string]pd.Instance, error) {
	return w.infra.NodePoolECS(*np)
}

type Triple struct {
	getter    TripleGetter
	role      string
	cluster   *api.Cluster
	mCRDs     []api.Master
	mnodes    []v1.Node
	instances []pd.Instance

	nodeInfo []NodeInfo
}

func (t *Triple) With(ins []pd.Instance) { t.instances = ins }

func (t *Triple) String() string {
	return fmt.Sprintf("[crd=%d, mnodes=%d, instance=%d]", len(t.mCRDs), len(t.mnodes), len(t.instances))
}

func (t *Triple) GetNode(id string) *v1.Node {
	for _, n := range t.mnodes {
		if strings.Contains(n.Spec.ProviderID, id) {
			return &n
		}
	}
	return nil
}

func NewTripleWorker(wgetter TripleGetter, np *api.NodePool) (*Triple, error) {
	trip := &Triple{role: pd.WorkerUserdata, getter: wgetter}
	spec, err := trip.getter.GetClusterItem()
	if err != nil {
		return trip, errors.Wrap(err, "get cluster")
	}
	trip.cluster = spec

	mNodes, err := trip.getter.GetWorkerNodeList(np)
	if err != nil {
		return trip, errors.Wrap(err, "master node")
	}
	trip.mnodes = mNodes
	detail, err := trip.getter.GetNodePoolECS(np)
	if err != nil {
		return nil, errors.Wrap(err, "ess master")
	}
	trip.instances = ToList(detail)

	for _, d := range detail {
		info := NodeInfo{Instance: &d, Role: trip.role}
		// 1. match node
		for _, n := range mNodes {
			nid := n.Spec.ProviderID
			if strings.Contains(nid, d.Id) {
				info.Node = &n
				break
			}
		}
		trip.nodeInfo = append(trip.nodeInfo, info)
	}

	// append extra mnodes
	for _, n := range mNodes {
		// 1. match nodeinfo
		ifound := false
		for _, i := range trip.nodeInfo {
			if i.Instance == nil {
				continue
			}
			nid := n.Spec.ProviderID
			if strings.Contains(nid, i.Instance.Id) {
				ifound = true
				break
			}
		}
		if !ifound {
			info := NodeInfo{Node: &n, Role: trip.role}
			trip.nodeInfo = append(trip.nodeInfo, info)
		}
	}
	return trip, nil
}

func NewTripleWorkerB(
	m client.Client, infra Infra, np *api.NodePool,
) (*Triple, error) {

	trip := &Triple{role: pd.WorkerUserdata}
	spec, err := h.Cluster(m, api.KUBERNETES_CLUSTER)
	if err != nil {
		return trip, errors.Wrap(err, "get cluster")
	}
	trip.cluster = spec

	mNodes, err := h.NodePoolItems(m, np)
	if err != nil {
		return trip, errors.Wrap(err, "master node")
	}
	trip.mnodes = mNodes
	detail, err := infra.NodePoolECS(*np)
	if err != nil {
		return nil, errors.Wrap(err, "ess master")
	}
	trip.instances = ToList(detail)

	for _, d := range detail {
		info := NodeInfo{Instance: &d, Role: trip.role}
		// 1. match node
		for _, n := range mNodes {
			nid := n.Spec.ProviderID
			if strings.Contains(nid, d.Id) {
				info.Node = &n
				break
			}
		}
		trip.nodeInfo = append(trip.nodeInfo, info)
	}

	// append extra mnodes
	for _, n := range mNodes {
		// 1. match nodeinfo
		ifound := false
		for _, i := range trip.nodeInfo {
			nid := n.Spec.ProviderID
			if strings.Contains(nid, i.Instance.Id) {
				ifound = true
				break
			}
		}
		if !ifound {
			info := NodeInfo{Node: &n, Role: trip.role}
			trip.nodeInfo = append(trip.nodeInfo, info)
		}
	}
	return trip, nil
}
func NewTripleMaster(getter TripleGetter) (*Triple, error) {

	trip := &Triple{role: pd.JoinMasterUserdata, getter: getter}
	spec, err := trip.getter.GetClusterItem()
	if err != nil {
		return trip, errors.Wrap(err, "get cluster")
	}
	trip.cluster = spec

	mCrds, err := trip.getter.GetMasterCR()
	if err != nil {
		return trip, errors.Wrap(err, "get master")
	}
	trip.mCRDs = mCrds

	mNodes, err := trip.getter.GetMasterNodeList()
	if err != nil {
		return trip, errors.Wrap(err, "master node")
	}
	trip.mnodes = mNodes
	detail, err := trip.getter.GetControlPlaneECS()
	if err != nil {
		return nil, errors.Wrap(err, "ess master")
	}
	trip.instances = ToList(detail)

	for _, d := range detail {
		i := d
		info := NodeInfo{Instance: &i, Role: trip.role}
		// 1. match mcrds
		for _, m := range mCrds {
			mcrd := m
			if strings.Contains(mcrd.Spec.ID, i.Id) {
				info.Resource = &mcrd
				break
			}
		}
		// 2. match node
		for _, n := range mNodes {
			node := n
			nid := node.Spec.ProviderID
			if strings.Contains(nid, i.Id) {
				info.Node = &node
				break
			}
		}
		trip.nodeInfo = append(trip.nodeInfo, info)
	}

	// append extra mcrds
	for _, m := range mCrds {
		k := m
		// 1. match nodeinfo
		ifound := false
		for _, i := range trip.nodeInfo {
			if i.Instance != nil {
				if strings.Contains(k.Spec.ID, i.Instance.Id) {
					ifound = true
					break
				}
			}
			if i.Node != nil {
				if strings.Contains(i.Node.Spec.ProviderID, k.Spec.ID) {
					ifound = true
					break
				}
			}
		}
		if !ifound {
			info := NodeInfo{Resource: &k, Role: trip.role}
			for _, n := range mNodes {
				node := n
				nid := n.Spec.ProviderID
				if strings.Contains(nid, k.Spec.ID) {
					info.Node = &node
					break
				}
			}
			trip.nodeInfo = append(trip.nodeInfo, info)
		}
	}

	// append extra mnodes
	for _, n := range mNodes {
		node := n
		// 1. match nodeinfo
		ifound := false
		for _, i := range trip.nodeInfo {
			nid := node.Spec.ProviderID
			if i.Instance != nil {
				if strings.Contains(nid, i.Instance.Id) {
					ifound = true
					break
				}
			}
			if i.Resource != nil {
				if strings.Contains(nid, i.Resource.Spec.ID) {
					ifound = true
					break
				}
			}
		}
		if !ifound {
			info := NodeInfo{Node: &node}
			for _, m := range mCrds {
				mcrd := m
				nid := node.Spec.ProviderID
				if strings.Contains(nid, m.Spec.ID) {
					info.Resource = &mcrd
					break
				}
			}
			trip.nodeInfo = append(trip.nodeInfo, info)
		}
	}
	klog.Infof("build trip: %s", trip.nodeInfo)
	return trip, nil
}

func NewTripleMasterB(m client.Client, infra Infra) (*Triple, error) {

	trip := &Triple{role: pd.JoinMasterUserdata}
	spec, err := h.Cluster(m, api.KUBERNETES_CLUSTER)
	if err != nil {
		return trip, errors.Wrap(err, "get cluster")
	}
	trip.cluster = spec

	mCrds, err := h.MasterCRDS(m)
	if err != nil {
		return trip, errors.Wrap(err, "get master")
	}
	trip.mCRDs = mCrds

	mNodes, err := h.MasterNodes(m)
	if err != nil {
		return trip, errors.Wrap(err, "master node")
	}
	trip.mnodes = mNodes
	detail, err := infra.ControlPlaneECS()
	if err != nil {
		return nil, errors.Wrap(err, "ess master")
	}
	trip.instances = ToList(detail)

	for _, d := range detail {
		info := NodeInfo{Instance: &d, Role: trip.role}
		// 1. match mcrds
		for _, m := range mCrds {
			if m.Spec.ID == d.Id {
				info.Resource = &m
				break
			}
		}
		// 2. match node
		for _, n := range mNodes {
			nid := n.Spec.ProviderID
			if strings.Contains(nid, d.Id) {
				info.Node = &n
				break
			}
		}
		trip.nodeInfo = append(trip.nodeInfo, info)
	}

	// append extra mcrds
	for _, m := range mCrds {
		// 1. match nodeinfo
		ifound := false
		for _, i := range trip.nodeInfo {
			if m.Spec.ID == i.Instance.Id {
				ifound = true
				break
			}
		}
		if !ifound {
			info := NodeInfo{Resource: &m, Role: trip.role}
			for _, n := range mNodes {
				nid := n.Spec.ProviderID
				if strings.Contains(nid, m.Spec.ID) {
					info.Node = &n
					break
				}
			}
			trip.nodeInfo = append(trip.nodeInfo, info)
		}
	}

	// append extra mnodes
	for _, n := range mNodes {
		// 1. match nodeinfo
		ifound := false
		for _, i := range trip.nodeInfo {
			nid := n.Spec.ProviderID
			if strings.Contains(nid, i.Instance.Id) {
				ifound = true
				break
			}
		}
		if !ifound {
			info := NodeInfo{Node: &n}
			for _, m := range mCrds {
				nid := n.Spec.ProviderID
				if strings.Contains(nid, m.Spec.ID) {
					info.Resource = &m
					break
				}
			}
			trip.nodeInfo = append(trip.nodeInfo, info)
		}
	}
	return trip, nil
}

func (t *Triple) SystemInConsistency() bool {
	uready := t.UnReadyNodeList()
	uiadd, uidel := t.InstanceCRDiff()
	ucadd, ucdel := t.InstanceNodeDiff()
	umadd, umdel := t.MasterCRDDiff()
	addl := len(uready) + len(uiadd) + len(uidel) + len(ucadd) + len(ucdel) + len(umadd) + len(umdel)
	return addl == 0
}

func (t *Triple) UnReadyNodeList() []NodeInfo {
	var infos []NodeInfo
	for _, n := range t.nodeInfo {
		if n.Node == nil || n.Instance == nil {
			continue
		}
		if !h.NodeReady(n.Node) {
			infos = append(infos, n)
		}
	}
	return infos
}

func (t *Triple) EtcdMemDiff(mems []etcd.Member) []etcd.Member {
	var deletions []etcd.Member
	for _, m := range mems {
		found := false
		for _, e := range t.instances {
			if m.IP == e.Ip {
				found = true
				break
			}
		}
		if !found {
			deletions = append(deletions, m)
		}
	}
	if float32(len(deletions)) > 0.5*float32(len(mems)) {
		klog.Warningf("can not remove major count of etcd member: "+
			"cluster might in an inconsistency state.[members=%d],[remove=%d]", len(mems), len(deletions))
		return []etcd.Member{}
	}
	return deletions
}

// InstanceCRDiff deprecated
func (t *Triple) InstanceCRDiff() ([]NodeInfo, []NodeInfo) {
	var (
		deletion []NodeInfo
		addition []NodeInfo
	)
	for _, i := range t.nodeInfo {
		if i.Instance != nil &&
			i.Resource == nil {
			addition = append(addition, i)
		}
		if i.Instance == nil &&
			i.Resource != nil {
			deletion = append(deletion, i)
		}
	}
	// return additions, deletions
	return addition, deletion
}

func (t *Triple) InstanceNodeDiff() ([]NodeInfo, []NodeInfo) {

	var (
		deletion []NodeInfo
		addition []NodeInfo
	)
	for _, i := range t.nodeInfo {
		if i.Instance != nil &&
			i.Node == nil {
			addition = append(addition, i)
		}
		if i.Instance == nil &&
			i.Node != nil {
			deletion = append(deletion, i)
		}
	}
	// additions instance with no correspond Node object, do reset fix.
	// deletions Node with no correspond instance. do remove Node object
	return addition, deletion
}

func BreakID(id string) (string, string, error) {
	pid := strings.Split(id, ".")
	if len(pid) != 2 {
		return "", "", fmt.Errorf("format error: %s", id)
	}
	// region.id
	return pid[0], pid[1], nil
}

func (t *Triple) MasterCRDDiff() ([]NodeInfo, []NodeInfo) {
	// master instance will be automatically aligned
	var (
		tdel   []NodeInfo
		missed []NodeInfo
	)
	for _, i := range t.nodeInfo {
		if i.Resource != nil {
			if i.Instance != nil {
				// instance still exist, do not delete crd
				continue
			}
			tdel = append(tdel, i)
		}
		if i.Resource == nil &&
			i.Node != nil {
			if i.Instance == nil {
				// instance is missing. do not rebuild master crd
				continue
			}
			missed = append(missed, i)
		}
	}
	// tdel: to be deleted
	// missed  : to be reconstructed
	return tdel, missed
}
