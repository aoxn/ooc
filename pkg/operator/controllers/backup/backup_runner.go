package backup

import (
	"context"
	"fmt"
	"github.com/aoxn/ooc/pkg/actions/etcd"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	h "github.com/aoxn/ooc/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewSnapshot(
	prvd provider.Interface,
) *Snapshot {
	recon := &Snapshot{ prvd: prvd, lock: &sync.RWMutex{}}
	return recon
}

var _ manager.Runnable = &Snapshot{}

type Snapshot struct {
	lock   *sync.RWMutex
	prvd   provider.Interface
	cache  cache.Cache
	client client.Client
	index  *h.Index
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
	if err := s.initialize();err != nil {
		return errors.Wrap(err, "start snapshot")
	}
	klog.Info("snapshot index data: ")
	fmt.Printf(s.index.String())
	go wait.Forever(s.CleanUp, 10 * time.Minute)
	klog.Infof("snapshot controller started... ")

	backup := func() {
		err := s.doBackup()
		if err != nil {
			klog.Errorf("backup etcd: %s", err.Error())
		}
	}
	wait.Forever(backup, 10 *time.Minute)
	return nil
}

func (s *Snapshot) initialize() error {
	spec, err := h.Cluster(s.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return fmt.Errorf("member: spec not found,%s", err.Error())
	}

	s.index = h.NewIndex(spec.Spec.ClusterID)
	oc, err := s.prvd.GetObject(s.index.IndexLocation())
	if err != nil {
		if strings.Contains(
			err.Error(), "NoSuchKey",
		) {
			// empty backup index information
			klog.Infof("no index.json found, default empty")
			return nil
		}
		return errors.Wrapf(err, "get index for %s", spec.Spec.ClusterID)
	}
	return s.index.LoadIndex(oc)
}

const KEEP_COPIES_CNT = 4

func (s *Snapshot) CleanUp() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(s.index.Copies) <= KEEP_COPIES_CNT {
		return
	}
	s.index.SortBackups()
	deleted := false
	var bck []h.Backup
	for k, backup := range s.index.Copies {
		if k < KEEP_COPIES_CNT {
			bck = append(bck, backup)
			continue
		}
		deleted = true
		err := s.prvd.DeleteObject(s.index.Path(backup))
		klog.Infof("remove etcd backup copies: %s, %v",s.index.Path(backup), err)
	}
	if deleted {
		s.index.Copies = bck
		err := s.prvd.PutObject(s.index.Bytes(),s.index.IndexLocation())
		if err != nil {
			klog.Errorf("clean up, put index object fail: %s", err.Error())
		}
	}
	klog.Infof("clean up backups: %d", len(s.index.Copies))
	return
}


func (s *Snapshot) doBackup() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	masters, err := h.Masters(s.client)
	if err != nil {
		return fmt.Errorf("member master: %s", err.Error())
	}
	spec, err := h.Cluster(s.client, api.KUBERNETES_CLUSTER)
	if err != nil {
		return fmt.Errorf("member: spec not found,%s", err.Error())
	}

	metcd, err := etcd.NewEtcdFromMasters(masters, spec, etcd.ETCD_TMP)
	if err != nil {
		return fmt.Errorf("new etcd: %s", err.Error())
	}
	now := h.HourNow()
	src := filepath.Join("/tmp",now,"snapshot.db")
	err = metcd.Snapshot(src)
	if err != nil {
		return errors.Wrap(err, "snapshot etcd")
	}
	backup := h.Backup{Identity: h.HourNow()}
	klog.Infof("trying to backup etcd to oss: [%s]", s.index.Path(backup))
	err = s.prvd.PutFile(src, s.index.Path(backup))
	if err != nil {
		return errors.Wrapf(err, "put file: %s", src)
	}
	s.index.Copies = append(s.index.Copies, backup)
	s.index.Spec   = spec
	err = s.prvd.PutObject(s.index.Bytes(),s.index.IndexLocation())
	if err != nil {
		return errors.Wrapf(err, "put index object: %s", src)
	}
	klog.Infof("backup etcd finished: %s", s.index.Path(backup))
	return nil
}
