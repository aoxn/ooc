package index

import (
	"encoding/json"
	"fmt"
	api "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	KEEP_COPIES_CNT = 4
	SnapshotTMP     = "/tmp/snapshot.db"
)

func NewSnapshotIndex(
	id string, store pd.ObjectStorage,
) *SnapshotIndex {
	return &SnapshotIndex{
		load:     false,
		store:    store,
		snapshot: newSnapshot(id),
	}
}

type SnapshotIndex struct {
	load     bool
	lock     sync.RWMutex
	snapshot *Snapshot
	store    pd.ObjectStorage
}

func (i *SnapshotIndex) LazyLoad() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.load {
		return nil
	}
	if i.snapshot == nil {
		return fmt.Errorf("snapshot not initialized: %v", i.snapshot)
	}
	location := i.snapshot.IndexLocation()
	data, err := i.store.GetObject(location)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			klog.Infof("no backup copy found, initial a new one")
			return nil
		}
		return errors.Wrapf(err, "get snapshot: %s", location)
	}
	return i.snapshot.Load(data)
}

func (i *SnapshotIndex) Snapshot() (*Snapshot, error) { return i.snapshot, i.LazyLoad() }

func (i *SnapshotIndex) BootSpec() (*api.ClusterSpec, error) { return i.snapshot.Spec, i.LazyLoad() }

func (i *SnapshotIndex) LatestBackup(dir string) (*api.ClusterSpec, error) {
	if err := i.LazyLoad(); err != nil {
		return nil, errors.Wrapf(err, "load latest backup")
	}
	i.lock.Lock()
	defer i.lock.Unlock()

	if dir == "" {
		dir = SnapshotTMP
	}
	backup := i.snapshot.LatestBackup()
	if backup == nil {
		return i.snapshot.Spec, fmt.Errorf("BackupNotFound")
	}
	err := i.store.GetFile(i.snapshot.Path(*backup), dir)
	if err != nil {
		return nil, errors.Wrapf(err, "download latest backup")
	}

	return i.snapshot.Spec, nil
}

func (i *SnapshotIndex) Backup(id api.ClusterSpec) error {
	if err := i.LazyLoad(); err != nil {
		return errors.Wrapf(err, "load latest backup")
	}
	i.lock.Lock()
	defer i.lock.Unlock()

	backup := Backup{Identity: HourNow()}
	klog.Infof("trying to backup etcd to oss: [%s]", i.snapshot.Path(backup))
	err := i.store.PutFile(SnapshotTMP, i.snapshot.Path(backup))
	if err != nil {
		return errors.Wrapf(err, "put file %s: %s", SnapshotTMP, i.snapshot.Path(backup))
	}
	i.snapshot.Copies = append(i.snapshot.Copies, backup)
	i.snapshot.Spec = &id
	err = i.store.PutObject(i.snapshot.Bytes(), i.snapshot.IndexLocation())
	if err != nil {
		return errors.Wrapf(err, "put snapshot object: %s", i.snapshot.IndexLocation())
	}
	klog.Infof("backup etcd finished: %s", i.snapshot.Path(backup))
	return nil
}

func (i *SnapshotIndex) BackupGC() error {
	if err := i.LazyLoad(); err != nil {
		return errors.Wrapf(err, "load latest backup")
	}
	i.lock.Lock()
	defer i.lock.Unlock()

	if len(i.snapshot.Copies) <= KEEP_COPIES_CNT {
		return nil
	}
	i.snapshot.SortBackups()
	deleted := false
	var bck []Backup
	for k, backup := range i.snapshot.Copies {
		if k < KEEP_COPIES_CNT {
			bck = append(bck, backup)
			continue
		}
		deleted = true
		err := i.store.DeleteObject(i.snapshot.Path(backup))
		klog.Infof("remove etcd backup copies: %s, %v", i.snapshot.Path(backup), err)
	}
	if deleted {
		i.snapshot.Copies = bck
		err := i.store.PutObject(i.snapshot.Bytes(), i.snapshot.IndexLocation())
		if err != nil {
			klog.Errorf("clean up, put snapshot object fail: %i", err.Error())
		}
	}
	klog.Infof("clean up backups: %d", len(i.snapshot.Copies))
	return nil
}

func NewSnapshotFrom(data []byte) (Snapshot, error) {
	i := Snapshot{}
	return i, i.Load(data)
}

func newSnapshot(name string) *Snapshot {
	return &Snapshot{
		Prefix: "wdrip/backup",
		Name:   name,
	}
}

type Snapshot struct {
	Prefix string           `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Name   string           `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	Copies []Backup         `json:"copies,omitempty" protobuf:"bytes,3,opt,name=copies"`
	Spec   *api.ClusterSpec `json:"spec,omitempty" protobuf:"bytes,4,opt,name=spec"`
}

type Backup struct {
	Identity string `json:"identity,omitempty" protobuf:"bytes,1,opt,name=identity"`
}

func (i *Snapshot) base() string {
	return fmt.Sprintf("%s/%s", i.Prefix, i.Name)
}
func (i *Snapshot) IndexLocation() string {
	return fmt.Sprintf("%s/index.json", i.base())
}

func (i *Snapshot) URI(b Backup, bName string) string {
	return fmt.Sprintf("oss://%s/%s", bName, i.Path(b))
}

func (i *Snapshot) Path(b Backup) string {
	return fmt.Sprintf("%s/%s/snapshot.db", i.base(), b.Identity)
}

func (i *Snapshot) Load(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, i)
}

func (i *Snapshot) String() string { return utils.PrettyJson(i) }

func (i *Snapshot) Bytes() []byte { return []byte(utils.PrettyJson(i)) }

func (i *Snapshot) SortBackups() {
	cmp := func(m, n int) bool {
		return i.Copies[m].Identity > i.Copies[n].Identity
	}
	sort.Slice(i.Copies, cmp)
}

func (i *Snapshot) LatestBackup() *Backup {
	i.SortBackups()
	if len(i.Copies) <= 0 {
		return nil
	}
	return &i.Copies[0]
}

func HourNow() string {
	return time.Now().Format("20060102-1504")
}
