package help

import (
	"context"
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/iaas/provider/alibaba"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

func After(
	t string, duration time.Duration,
) bool {
	me, err := time.ParseInLocation("2006-01-02T15:04:05Z", t, time.Local)
	if err != nil {
		klog.Infof("parse time error: %s", err.Error())
		return false
	}
	return time.Now().After(me.Add(duration))
}

func Now() string {
	return time.Now().Format("2006-01-02T15:04:05Z")
}

func WaitMasterReady(
	cclient client.Client, id string,
) error {
	return wait.Poll(
		5*time.Second,
		4*time.Minute,
		func() (done bool, err error) {
			klog.Infof("[WaitMasterReady] wait master ready: %s", id)

			nodes, err := MasterNodes(cclient)
			if err != nil {
				klog.Warningf("[WaitMasterReady] get master nodes: %s", err.Error())
				return false, nil
			}
			for _, n := range nodes {
				if strings.Contains(n.Spec.ProviderID, id) {
					return NodeReady(&n), nil
				}
			}
			return false, nil
		},
	)
}

func LoadStack(
	prvd provider.Interface,
	ctx *provider.Context,
	spec *api.Cluster,
) (map[string]provider.Value, error) {
	if spec.Spec.Bind.ResourceId == "" {
		resource, err := prvd.GetStackOutPuts(
			provider.NewContextWithCluster(&spec.Spec),
			&api.ClusterId{ObjectMeta: metav1.ObjectMeta{Name: spec.Spec.ClusterID}},
		)
		if err != nil {
			return nil, fmt.Errorf("provider: list resource fail, %s", err.Error())
		}
		spec.Spec.Bind.ResourceId = resource[alibaba.StackID].Val.(string)
	}
	id := &api.ClusterId{
		Spec: api.ClusterIdSpec{
			ResourceId: spec.Spec.Bind.ResourceId,
		},
	}
	return prvd.GetInfraStack(ctx, id)
}

func Cluster(
	rclient client.Client,
	name string,
) (*api.Cluster, error) {
	cluster := &api.Cluster{}
	err := rclient.Get(
		context.TODO(),
		client.ObjectKey{Name: name},
		cluster,
	)
	return cluster, err
}

func MyCluster(
	me client.Client,
) (*api.ClusterSpec, error) {
	//
	cluster := &api.Cluster{}
	err := me.Get(
		context.TODO(),
		client.ObjectKey{
			Namespace: "kube-system",
			Name:      "kubernetes-cluster",
		},
		cluster,
	)
	return &cluster.Spec, err
}

func MasterSet(
	rclient client.Client,
	name string,
) (*api.MasterSet, error) {
	cluster := &api.MasterSet{}
	err := rclient.Get(
		context.TODO(),
		client.ObjectKey{Name: name},
		cluster,
	)
	return cluster, err
}

func HasNodePoolID(node v1.Node, npid string) bool {
	if node.Labels == nil {
		return false
	}
	id,ok := node.Labels["np.ovm.io/id"]
	if !ok {
		return false
	}
	return id == npid
}

func NodePoolItems(
	rclient client.Client, np *api.NodePool,
) ([]v1.Node, error) {
	require, _ := labels.NewRequirement(
		"np.ovm.io/id", "=", []string{np.Spec.NodePoolID},
	)
	mnode := &v1.NodeList{}
	err := rclient.List(
		context.TODO(),
		mnode,
		&client.ListOptions{
			LabelSelector: labels.NewSelector().Add(*require),
		},
	)
	return mnode.Items, err
}


func MasterNodes(
	rclient client.Client,
) ([]v1.Node, error) {
	require, _ := labels.NewRequirement(
		"node-role.kubernetes.io/master", "=", []string{""},
	)
	mnode := &v1.NodeList{}
	err := rclient.List(
		context.TODO(),
		mnode,
		&client.ListOptions{
			LabelSelector: labels.NewSelector().Add(*require),
		},
	)
	return mnode.Items, err
}

func Workers(
	rclient client.Client,
) ([]v1.Node, error) {
	nodes, err := NodeItems(rclient)
	if err != nil {
		return nodes, err
	}
	var mnode []v1.Node
	for _, n := range nodes {
		if !IsMaster(&n) {
			mnode = append(mnode, n)
		}
	}
	return mnode, nil
}

func NodeItems(
	rclient client.Client,
) ([]v1.Node, error) {
	mnode := &v1.NodeList{}
	err := rclient.List(
		context.TODO(),
		mnode,
		&client.ListOptions{},
	)
	return mnode.Items, err
}

func MasterCRDS(
	rclient client.Client,
) ([]api.Master, error) {
	masters := &api.MasterList{}
	err := rclient.List(
		context.TODO(), masters,
	)
	return masters.Items, err
}

func Master(
	rclient client.Client,
	name string,
) (*api.Master, error) {
	master := &api.Master{}
	err := rclient.Get(
		context.TODO(), client.ObjectKey{Name: name}, master,
	)
	return master, err
}


func Node(
	rclient client.Client,
	name string,
) (*v1.Node, error) {
	node := &v1.Node{}
	err := rclient.Get(
		context.TODO(), client.ObjectKey{Name: name}, node,
	)
	return node, err
}

func IsMaster(node *v1.Node) bool {
	lbl := node.Labels
	if lbl == nil {
		lbl = make(map[string]string)
	}
	_, ok := lbl["node-role.kubernetes.io/master"]
	return ok
}

func Condition(
	node v1.Node,
) *v1.NodeCondition {
	for _, v := range node.Status.Conditions {
		if v.Type == "Ready" {
			return &v
		}
	}
	return nil
}

func NewDelay(i int64) reconcile.Result {
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Duration(i * int64(time.Second)),
	}
}

func Has(m []string, tar string) bool {
	for _, v := range m {
		if v == tar {
			return true
		}
	}
	return false
}

func Remove(m []string, tar string) []string {
	var result []string
	for _, v := range m {
		if v == tar {
			continue
		}
		result = append(result, v)
	}
	return result
}

func Max(
	x, y int,
) int {
	if x < y {
		return y
	}
	return x
}

func InCluster() bool { return os.Getenv("KUBERNETES_SERVICE_HOST") != "" }

func MasterNames(
	nodes []api.Master,
) []string {
	var crds []string
	for _, n := range nodes {
		crds = append(crds, n.Name)
	}
	return crds
}

func NodeNames(
	nodes []v1.Node,
) []string {
	var mnode []string
	for _, n := range nodes {
		mnode = append(mnode, n.Name)
	}
	return mnode
}

func ECSNames(
	ecss map[string]provider.Instance,
) []string {
	var ecs []string
	for _, v := range ecss {
		ecs = append(ecs, v.Ip)
	}
	return ecs
}

func HasMasterRole(label map[string]string) bool {
	_, ok := label["node-role.kubernetes.io/master"]
	return ok
}
