package heal

import (
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var (
	node1 = v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "192.168.0.103.i-bp1c65aivl1e4vkm9e2m",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Spec: v1.NodeSpec{
			ProviderID: "cn-hangzhou.i-bp1c65aivl1e4vkm9e2m",
		},
	}

	node2 = v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "192.168.0.104.i-bp1c65aivl1e4vkm9e2n",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Spec: v1.NodeSpec{
			ProviderID: "cn-hangzhou.i-bp1c65aivl1e4vkm9e2n",
		},
	}

	node3 = v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "192.168.0.105.i-bp1c65aivl1e4vkm9e2k",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Spec: v1.NodeSpec{
			ProviderID: "cn-hangzhou.i-bp1c65aivl1e4vkm9e2k",
		},
	}

	master1 = api.Master{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cn-hangzhou.i-bp1c65aivl1e4vkm9e2m",
		},
		Spec: api.MasterSpec{
			IP:   "192.168.0.103",
			Role: "Hybrid",
			ID:   "cn-hangzhou.i-bp1c65aivl1e4vkm9e2m",
		},
	}

	master2 = api.Master{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cn-hangzhou.i-bp1c65aivl1e4vkm9e2n",
		},
		Spec: api.MasterSpec{
			IP:   "192.168.0.104",
			Role: "Hybrid",
			ID:   "cn-hangzhou.i-bp1c65aivl1e4vkm9e2n",
		},
	}

	master3 = api.Master{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cn-hangzhou.i-bp1c65aivl1e4vkm9e2k",
		},
		Spec: api.MasterSpec{
			IP:   "192.168.0.105",
			Role: "Hybrid",
			ID:   "cn-hangzhou.i-bp1c65aivl1e4vkm9e2k",
		},
	}

	inst1 = provider.Instance{
		Region: "cn-hangzhou",
		Id:     "i-bp1c65aivl1e4vkm9e2m",
		Ip:     "192.168.0.103",
		Status: "running",
	}

	inst2 = provider.Instance{
		Region: "cn-hangzhou",
		Id:     "i-bp1c65aivl1e4vkm9e2n",
		Ip:     "192.168.0.104",
		Status: "running",
	}

	inst3 = provider.Instance{
		Region: "cn-hangzhou",
		Id:     "i-bp1c65aivl1e4vkm9e2k",
		Ip:     "192.168.0.105",
		Status: "running",
	}
)

type fakeTripleGetter struct {
	masterNodes []v1.Node
	workerNodes []v1.Node

	spec *api.Cluster

	masterCRDs       []api.Master
	masterInstances  map[string]provider.Instance
	nodepoolInstance map[string]provider.Instance
}

func (f fakeTripleGetter) GetClusterItem() (*api.Cluster, error) { return f.spec, nil }

func (f fakeTripleGetter) GetMasterNodeList() ([]v1.Node, error) { return f.masterNodes, nil }

func (f fakeTripleGetter) GetWorkerNodeList(np *api.NodePool) ([]v1.Node, error) {
	return f.workerNodes, nil
}

func (f fakeTripleGetter) GetMasterCR() ([]api.Master, error) { return f.masterCRDs, nil }

func (f fakeTripleGetter) GetControlPlaneECS() (map[string]provider.Instance, error) {
	return f.masterInstances, nil
}

func (f fakeTripleGetter) GetNodePoolECS(np *api.NodePool) (map[string]provider.Instance, error) {
	return f.nodepoolInstance, nil
}

func TestConsistency1(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1},
		masterCRDs:      []api.Master{master1},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.True(t, trip.SystemInConsistency())
}

func TestConsistency2(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master1, master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1, inst2.Id: inst2},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.True(t, trip.SystemInConsistency())
}

func TestInConsistency1(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1},
		masterCRDs:      []api.Master{master1, master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.False(t, trip.SystemInConsistency())
}

func TestInConsistency2(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master1, master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.False(t, trip.SystemInConsistency())
}

func TestInConsistency3(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master1},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.False(t, trip.SystemInConsistency())
}

func TestInConsistency4(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.False(t, trip.SystemInConsistency())
}

func TestInConsistency5(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{},
		masterInstances: map[string]provider.Instance{},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.False(t, trip.SystemInConsistency())
}

func TestInConsistency6(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master1, master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1, inst2.Id: inst2, inst3.Id: inst3},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.False(t, trip.SystemInConsistency())
}

func TestCRD1(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1, inst2.Id: inst2, inst3.Id: inst3},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	tdel, missed := trip.MasterCRDDiff()
	if len(missed) != 2 || len(tdel) != 0 {
		t.Fatalf("master crd unexpected")
	}
	ids := missed[0].Node.Name + "." + missed[1].Node.Name
	assert.Contains(t, ids, node1.Name)
	assert.Contains(t, ids, node2.Name)
}

func TestCRD2(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	tdel, missed := trip.MasterCRDDiff()
	if len(missed) != 1 || len(tdel) != 0 {
		t.Fatalf("master crd unexpected")
	}
	ids := missed[0].Node.Name
	assert.Contains(t, ids, node1.Name)
}

func TestCRD3(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	tdel, missed := trip.MasterCRDDiff()
	t.Logf("missed: %v", missed)
	t.Logf("tdel  : %v", tdel)
	if len(missed) != 1 || len(tdel) != 1 {
		t.Fatalf("master crd unexpected")
	}
	ids := missed[0].Node.Name
	assert.Contains(t, ids, node1.Name)

	assert.Contains(t, tdel[0].Node.Name, node2.Name)
}

func TestInstanceNode1(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	addition, deletion := trip.InstanceNodeDiff()
	t.Logf("deletion: %v", deletion)
	t.Logf("addition: %v", addition)
	if len(deletion) != 1 || len(addition) != 0 {
		t.Fatalf("instance node test unexpected")
	}

	assert.Contains(t, deletion[0].Node.Name, node2.Name)
}

func TestInstanceNode2(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1, inst2.Id: inst2},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	addition, deletion := trip.InstanceNodeDiff()
	t.Logf("deletion: %v", deletion)
	t.Logf("addition: %v", addition)
	if len(deletion) != 0 || len(addition) != 0 {
		t.Fatalf("instance node test unexpected")
	}
}

func TestInstanceNode3(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node2},
		masterCRDs:      []api.Master{master2},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1, inst2.Id: inst2, inst3.Id: inst3},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	addition, deletion := trip.InstanceNodeDiff()
	t.Logf("deletion: %v", deletion)
	t.Logf("addition: %v", addition)
	if len(deletion) != 0 || len(addition) != 1 {
		t.Fatalf("instance node test unexpected")
	}

	assert.Contains(t, node3.Name, addition[0].Instance.Id)
}

func TestNewTripleMaster(t *testing.T) {
	getter := fakeTripleGetter{
		masterNodes:     []v1.Node{node1, node3},
		masterCRDs:      []api.Master{master2, master1},
		masterInstances: map[string]provider.Instance{inst1.Id: inst1},
	}
	trip, err := NewTripleMaster(getter)
	if err != nil {
		t.Fatalf("new triple master: %s", err.Error())
	}
	assert.Equal(t, len(trip.nodeInfo), 3)
}
