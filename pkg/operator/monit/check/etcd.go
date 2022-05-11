package check

import (
	"github.com/aoxn/wdrip/pkg/actions/etcd"
	api "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	h "github.com/aoxn/wdrip/pkg/operator/controllers/help"
	"github.com/aoxn/wdrip/pkg/operator/monit"
	"github.com/pkg/errors"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCheckEtcd(client client.Client, qps float32) (*CheckEtcd, error) {
	spec, err := h.Cluster(client, "kubernetes-cluster")
	if err != nil {
		return nil, err
	}
	flow := flowcontrol.NewTokenBucketRateLimiter(0.1, 1)
	return &CheckEtcd{
		Cluster: spec,
		limit:   flow,
		client:  client,
	}, nil
}

type CheckEtcd struct {
	monit.BaseCheck
	master  []api.Master
	Cluster *api.Cluster
	client  client.Client
	limit   flowcontrol.RateLimiter
}

func (m *CheckEtcd) Name() string { return "etcd.health.check" }

func (m *CheckEtcd) Check() (bool, error) {
	klog.Infof("begin to check etcd, [%s]", m.Name())
	spec, err := h.Cluster(m.client, "kubernetes-cluster")
	if err != nil {
		klog.Warningf("find my cluster failed: %s", err.Error())
	} else {
		// in case of apiserver down, use cached spec & apiserver
		m.Cluster = spec
		master, err := h.MasterCRDS(m.client)
		if err != nil || len(master) == 0 {
			klog.Warningf("find master failed: %v", err)
		} else {
			m.master = master
		}
		if len(m.master) == 0 {
			return true, errors.Wrapf(err, "no master crd found")
		}
		klog.Infof("etcd check try masters: %v", api.ToMasterStringList(m.master))
	}
	if m.Cluster == nil {
		return true, errors.Wrapf(err, "empty cluster spec")
	}
	metcd, err := etcd.NewEtcdFromCRD(m.master, m.Cluster, "/tmp")
	if err != nil {
		return true, errors.Wrap(err, "new etcd")
	}
	mems, err := metcd.MemberList()
	if err != nil {
		klog.Errorf("[FAILED]etcd check: %s", err.Error())
		return false, nil
	}
	for _, mem := range mems.Members {
		if err := metcd.EndpointHealth(mem.IP); err == nil {
			klog.Info("at least one member is health, skip recover")
			return true, nil
		}
	}
	klog.Infof("etcd check: all member enter unhealthy state %s", mems.Members)
	return false, nil
}

func (m *CheckEtcd) Limit() flowcontrol.RateLimiter { return m.limit }

func (m *CheckEtcd) Threshold() int { return 6 }
