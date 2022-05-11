package backup

import (
	"context"
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions/etcd"
	api "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	prvd "github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/aoxn/wdrip/pkg/index"
	h "github.com/aoxn/wdrip/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewSnapshot() *Snapshot {
	recon := &Snapshot{lock: &sync.RWMutex{}}
	return recon
}

func NewBareSnapshot(index *index.GenericIndexer) *Snapshot {
	return &Snapshot{
		index: index,
		lock:  &sync.RWMutex{},
	}
}

var _ manager.Runnable = &Snapshot{}

type Snapshot struct {
	spec   *api.Cluster
	lock   *sync.RWMutex
	cache  cache.Cache
	client client.Client
	index  *index.GenericIndexer
	//record event recorder
	record record.EventRecorder
}

func (s *Snapshot) InjectCache(cache cache.Cache) error {
	s.cache = cache
	return nil
}

func (s *Snapshot) InjectClient(me client.Client) error {
	s.client = me
	return nil
}

func (s *Snapshot) Start(ctx context.Context) error {
	klog.Infof("trying to start snapshot controller")
	if !s.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("wait for cache sync failed")
	}
	if err := s.initialize(); err != nil {
		return errors.Wrap(err, "start snapshot")
	}
	klog.Infof("snapshot index data: %s", s.spec.Spec.ClusterID)
	go wait.Forever(s.CleanUp, 10*time.Minute)
	klog.Infof("snapshot controller started... ")

	backup := func() {
		err := s.doBackup()
		if err != nil {
			klog.Errorf("backup etcd: %s", err.Error())
		}
	}
	wait.Forever(backup, 10*time.Minute)
	return nil
}

func (s *Snapshot) initialize() error {
	spec, err := h.Cluster(s.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return fmt.Errorf("member: spec not found,%s", err.Error())
	}
	s.spec = spec
	ctx, err := prvd.NewContext(&api.WdripOptions{}, &spec.Spec)
	if err != nil {
		return errors.Wrapf(err, "new provider context")
	}
	s.index = index.NewGenericIndexer(spec.Spec.ClusterID, ctx.Provider())
	return err
}

func (s *Snapshot) CleanUp() {
	s.lock.Lock()
	defer s.lock.Unlock()

	klog.Infof("start gc backups: %s", s.spec.Spec.ClusterID)
	err := s.index.BackupGC()
	if err != nil {
		klog.Errorf("gc backup fail: %s", err.Error())
	}
}

func (s *Snapshot) doBackup() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	masters, err := h.MasterCRDS(s.client)
	if err != nil {
		return fmt.Errorf("member master: %s", err.Error())
	}
	if len(masters) <= 0 {
		return fmt.Errorf("master crd not found %d, abort backup", len(masters))
	}
	spec, err := h.Cluster(s.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return fmt.Errorf("member: spec not found,%s", err.Error())
	}

	return s.Backup(spec, masters)
}

func (s *Snapshot) Backup(
	spec *api.Cluster,
	masters []api.Master,
) error {
	if err := s.initialize(); err != nil {
		return errors.Wrapf(err, "initialize snapshot backup failed")
	}
	metcd, err := etcd.NewEtcdFromCRD(masters, spec, etcd.ETCD_TMP)
	if err != nil {
		return fmt.Errorf("new etcd: %s", err.Error())
	}

	src := filepath.Join(index.SnapshotTMP)
	err = metcd.Snapshot(src)
	if err != nil {
		return errors.Wrap(err, "snapshot etcd")
	}
	mid, err := s.index.GetCluster(spec.Spec.ClusterID)
	if err != nil {
		return errors.Wrap(err, "load cluster id")
	}
	mid.Spec.Cluster = spec.Spec
	err = s.index.SaveCluster(mid)
	if err != nil {
		return errors.Wrapf(err, "save cluster spec")
	}
	return s.index.Backup(spec.Spec)
}
