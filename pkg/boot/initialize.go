package boot

import (
	"fmt"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

type Step func(ctx *context.NodeContext) error

/*
	Initialization

	Steps1: ContainerRuntime

	Steps2: ETCD

	Steps3: Kubernetes

	Steps4: WaitForEtcd

	Steps5: KubeadmInit
*/

func InitFunc(ctx *context.NodeContext) Step {
	flag := ctx.OocFlags()
	switch flag.BootType {
	case utils.BootTypeLocal:
		return InitFromCfg
	case utils.BootTypeCoord:
		return InitFromCoordinator
	case utils.BootTypeRecover:
		return InitFromRecover
	default:
		klog.Infof("default to %s", utils.BootTypeOperator)
	}
	return InitFromOperator
}

func InitFromOperator(ctx *context.NodeContext) error {
	client := ctx.BootClusterClient()
	var cluster *api.Cluster
	err := wait.PollImmediate(
		5*time.Second,
		5*time.Minute,
		func() (done bool, err error) {
			mc, err := client.Get("kubernetes-cluster")
			if err != nil {
				klog.Infof("trying to retrieve bootcfg: %s", err.Error())
				return false, nil
			}
			cluster = mc
			klog.Infof("retrieve bootcfg succeed")
			return true, nil
		},
	)
	if err != nil {
		return fmt.Errorf("retrieve bootcfg: %s", err.Error())
	}
	fmt.Println("bootcfg from operator: ", utils.PrettyYaml(cluster))
	meta := ctx.NodeMetaData()
	if meta == nil {
		return fmt.Errorf("nil client, meta: %v", meta)
	}
	id, err := meta.NodeID()
	if err != nil {
		return fmt.Errorf("meta data error node id: %s", err.Error())
	}

	ip, err := meta.NodeIP()
	if err != nil {
		return fmt.Errorf("meta data error node ip: %s", err.Error())
	}
	node := api.Master{
		ObjectMeta: metav1.ObjectMeta{
			Name: ip,
		},
		Spec: api.MasterSpec{
			ID:   id,
			IP:   ip,
			Role: ctx.OocFlags().Role,
		},
		Status: api.MasterStatus{
			Peer:    cluster.Status.Peers,
			BootCFG: cluster,
		},
	}
	AddExtraSans(cluster)
	ctx.SetKV(context.NodeInfoObject, &node)
	return nil
}

func AddExtraSans(spec *api.Cluster) {
	addSans := func(endpoint string) {
		found := false
		for _, ip := range spec.Spec.Sans {
			if ip == endpoint {
				found = true
				break
			}
		}
		if !found {
			spec.Spec.Sans = append(spec.Spec.Sans, endpoint)
			klog.Infof("append extra sans: %s", endpoint)
		}
	}
	if spec.Spec.Endpoint.Intranet != "" {
		addSans(spec.Spec.Endpoint.Intranet)
	}
	if spec.Spec.Endpoint.Internet != "" {
		addSans(spec.Spec.Endpoint.Internet)
	}
}

func InitFromCfg(ctx *context.NodeContext) error {

	cfg := ctx.OocFlags().Config
	if cfg == "" {
		return fmt.Errorf("empty cluster config, --config")
	}
	bcfg, err := ioutil.ReadFile(cfg)
	if err != nil {
		return err
	}
	cluster := &api.ClusterSpec{}
	err = yaml.Unmarshal(bcfg, cluster)
	if err != nil {
		return fmt.Errorf("error decode cluster: %s", err.Error())
	}
	klog.Infof("read ooc config from %s: %+v", cfg, utils.PrettyYaml(cluster))
	genf := fmt.Sprintf("%s.gen", cfg)
	exist, err := utils.FileExist(genf)
	if err != nil {
		return fmt.Errorf("read file stat: %s, %s", genf, err.Error())
	}
	if exist {
		klog.Infof("file %s exists, re-initialize. keep previous CA", genf)
		bcfg, err := ioutil.ReadFile(genf)
		if err != nil {
			return err
		}
		prec := &api.ClusterSpec{}
		err = yaml.Unmarshal(bcfg, prec)
		if err != nil {
			return fmt.Errorf("error decode cluster: %s", err.Error())
		}
		if prec.Kubernetes.ControlRoot != nil &&
			cluster.Kubernetes.ControlRoot == nil {
			cluster.Kubernetes.ControlRoot = prec.Kubernetes.ControlRoot
		}
		if prec.Kubernetes.FrontProxyCA != nil &&
			cluster.Kubernetes.FrontProxyCA == nil {
			cluster.Kubernetes.FrontProxyCA = prec.Kubernetes.FrontProxyCA
		}
		if prec.Kubernetes.RootCA != nil &&
			cluster.Kubernetes.RootCA == nil {
			cluster.Kubernetes.RootCA = prec.Kubernetes.RootCA
		}
		if prec.Kubernetes.SvcAccountCA != nil &&
			cluster.Kubernetes.SvcAccountCA == nil {
			cluster.Kubernetes.SvcAccountCA = prec.Kubernetes.SvcAccountCA
		}
		if prec.Etcd.PeerCA != nil &&
			cluster.Etcd.PeerCA == nil {
			cluster.Etcd.PeerCA = prec.Etcd.PeerCA
		}
		if prec.Etcd.ServerCA != nil &&
			cluster.Etcd.ServerCA == nil {
			cluster.Etcd.ServerCA = prec.Etcd.ServerCA
		}
	} else {
		klog.Infof("file %s not exist. initialize", genf)
	}
	meta := ctx.NodeMetaData()
	if meta == nil {
		return fmt.Errorf("nil client, meta: %v", meta)
	}
	id, err := meta.NodeID()
	if err != nil {
		return fmt.Errorf("meta data error node id: %s", err.Error())
	}

	ip, err := meta.NodeIP()
	if err != nil {
		return fmt.Errorf("meta data error node ip: %s", err.Error())
	}
	SetDefaultCredential(cluster)
	spec := api.NewDefaultCluster("kubernetes-cluster", *cluster)
	AddExtraSans(spec)
	node := api.Master{
		ObjectMeta: metav1.ObjectMeta{
			Name: ip,
		},
		Spec: api.MasterSpec{
			ID:   id,
			IP:   ip,
			Role: ctx.OocFlags().Role,
		},
		Status: api.MasterStatus{
			BootCFG: spec,
		},
	}
	ctx.SetKV(context.NodeInfoObject, &node)
	return nil
}

func InitFromRecover(ctx *context.NodeContext) error {

	/*
		Build an empty api.Master object
	*/
	meta := ctx.NodeMetaData()
	if meta == nil {
		return fmt.Errorf("nil client, meta: %v", meta)
	}
	id, err := meta.NodeID()
	if err != nil {
		return fmt.Errorf("meta data error node id: %s", err.Error())
	}

	ip, err := meta.NodeIP()
	if err != nil {
		return fmt.Errorf("meta data error node ip: %s", err.Error())
	}

	region, err := meta.Region()
	if err != nil {
		return fmt.Errorf("meta data error region: %s", err.Error())
	}

	opts := ctx.OocFlags()
	spec := api.NewRecoverCluster(opts.ClusterName, region, nil)
	node := api.Master{
		ObjectMeta: metav1.ObjectMeta{
			Name: ip,
		},
		Spec: api.MasterSpec{
			ID:   id,
			IP:   ip,
			Role: ctx.OocFlags().Role,
		},
		Status: api.MasterStatus{
			BootCFG: spec,
		},
	}
	// we use ~/.ooc/config to initializing provider
	pctx, err := provider.NewContext(&opts, &spec.Spec)
	if err != nil {
		return errors.Wrap(err, "initialize provider")
	}
	ctx.SetKV(context.ProviderCtx, pctx)
	mindex := pctx.Indexer()
	mspec, err := mindex.LatestBackup(spec.Spec.ClusterID, provider.SnapshotTMP)
	if err != nil {
		return errors.Wrap(err, "download backup db file")
	}
	node.Status.BootCFG = api.NewDefaultCluster("kubernetes-cluster", *mspec)
	ctx.SetKV(context.NodeInfoObject, &node)
	klog.Infof("read cluster config from oss backup %s: %+v", mindex.Index().IndexLocation(), utils.PrettyYaml(node))
	return nil
}

func InitFromCoordinator(ctx *context.NodeContext) error {
	err := RegisterMyself(ctx)
	if err != nil {
		return fmt.Errorf("regiter self node error: %s", err.Error())
	}
	return SetBootCredential(ctx)
}

// SetBootCredential help for cfg
func SetBootCredential(ctx *context.NodeContext) error {
	client := ctx.BootCredentialClient()
	meta := ctx.NodeMetaData()
	if meta == nil || client == nil {
		return fmt.Errorf("nil client, meta: %v, node:%v", meta, client)
	}
	id, err := meta.NodeID()
	if err != nil {
		return fmt.Errorf("meta data error setbootcredential: %s", err.Error())
	}
	mastercnt := ctx.ExpectedMasterCnt()
	node := &api.Master{}
	err = wait.Poll(
		2*time.Second,
		5*time.Minute,
		func() (done bool, err error) {
			node, err = client.Get(id)
			if err != nil {
				klog.Infof("retry waiting for credential error: %s", err.Error())
				return false, nil
			}
			if ctx.OocFlags().Role != api.NODE_ROLE_MASTER {
				klog.Infof("init worker node, %s", node.Spec.ID)
				return true, nil
			}
			if mastercnt != 0 &&
				len(node.Status.BootCFG.Spec.Etcd.Endpoints) < mastercnt {
				klog.Infof("wait for another %d master to register in...", mastercnt-len(node.Status.BootCFG.Spec.Etcd.Endpoints))
				return false, nil
			}
			return true, nil
		},
	)
	if err != nil {
		return fmt.Errorf("wait for credential config err: %s", err.Error())
	}
	klog.Infof("recieve NodeInfo: \n%s", utils.PrettyYaml(node))
	ctx.SetKV(context.NodeInfoObject, node)
	return nil
}

func RegisterMyself(ctx *context.NodeContext) error {
	client := ctx.BootNodeClient()
	meta := ctx.NodeMetaData()
	if meta == nil || client == nil {
		return fmt.Errorf("nil client, meta: %v, node:%v", meta, client)
	}
	id, err := meta.NodeID()
	if err != nil {
		return fmt.Errorf("meta data error nodeid: %s", err.Error())
	}
	ip, err := meta.NodeIP()
	if err != nil {
		return fmt.Errorf("meta data error nodeip: %s", err.Error())
	}

	return wait.Poll(
		2*time.Second,
		5*time.Minute,
		func() (done bool, err error) {
			_, err = client.Create(
				&api.Master{
					TypeMeta: v1.TypeMeta{
						Kind:       "NodeObject",
						APIVersion: "v1",
					},
					ObjectMeta: v1.ObjectMeta{
						Name: ip,
					},
					Spec: api.MasterSpec{
						ID:   id,
						IP:   ip,
						Role: ctx.OocFlags().Role,
					},
				},
			)
			if err != nil {
				klog.Infof("retry waiting for register myself error: %s", err.Error())
				return false, nil
			}
			return true, nil
		},
	)
}
