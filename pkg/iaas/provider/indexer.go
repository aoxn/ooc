package provider

import (
	"encoding/json"
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"sort"
	"strings"
	"sync"
	"time"
)

const KEEP_COPIES_CNT = 4

type Storage interface {
	ObjectStorage
	Save(id api.ClusterId) error
	Get(id string) (api.ClusterId, error)
	Remove(id string) error
	List(selector string) ([]api.ClusterId, error)
}

func NewIndexer(store Storage) *Indexer { return &Indexer{store: store} }

type Indexer struct {
	lock  sync.RWMutex
	store Storage
	index *BackupIndex
}

func (i *Indexer) LoadIndex(id string) error {
	i.index = NewIndex(id)
	data, err := i.store.GetObject(i.index.IndexLocation())
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			klog.Infof("no backup copy found, initial a new one")
			return nil
		}
		return errors.Wrapf(err, "get backup index: %s", i.index.IndexLocation())
	}
	return i.index.Load(data)
}

func (i *Indexer) Index() *BackupIndex { return i.index }

func (i *Indexer) Save(id api.ClusterId) error { return i.store.Save(id) }

func (i *Indexer) Get(id string) (api.ClusterId, error) { return i.store.Get(id) }

func (i *Indexer) Remove(id string) error { return i.store.Remove(id) }

func (i *Indexer) List(id string) ([]api.ClusterId, error) { return i.store.List(id) }

func (i *Indexer) ListBackups(id string) (*BackupIndex, error) {
	if id == "" {
		return nil, fmt.Errorf("cluster name must be specified")
	}
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.index == nil {
		err := i.LoadIndex(id)
		if err != nil {
			return nil, errors.Wrapf(err, "load index")
		}
	}
	return i.index, nil
}

func (i *Indexer) BootSpec(id string) (*api.ClusterSpec, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.index == nil {
		err := i.LoadIndex(id)
		if err != nil {
			return nil, errors.Wrapf(err, "load index")
		}
	}
	return i.index.Spec, nil
}

func (i *Indexer) LatestBackup(id, dir string) (*api.ClusterSpec, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.index == nil {
		err := i.LoadIndex(id)
		if err != nil {
			return nil, errors.Wrapf(err, "load index")
		}
	}
	if dir == "" {
		dir = SnapshotTMP
	}
	backup := i.index.LatestBackup()
	if backup == nil {
		return i.index.Spec, fmt.Errorf("BackupNotFound")
	}
	err := i.store.GetFile(i.index.Path(*backup), dir)
	if err != nil {
		return nil, errors.Wrapf(err, "download latest backup")
	}

	return i.index.Spec, nil
}

func (i *Indexer) Backup(id api.ClusterSpec) error {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.index == nil {
		err := i.LoadIndex(id.ClusterID)
		if err != nil {
			return errors.Wrapf(err, "load index")
		}
	}

	backup := Backup{Identity: HourNow()}
	klog.Infof("trying to backup etcd to oss: [%s]", i.index.Path(backup))
	err := i.store.PutFile(SnapshotTMP, i.index.Path(backup))
	if err != nil {
		return errors.Wrapf(err, "put file %s: %s", SnapshotTMP, i.index.Path(backup))
	}
	i.index.Copies = append(i.index.Copies, backup)
	i.index.Spec = &id
	err = i.store.PutObject(i.index.Bytes(), i.index.IndexLocation())
	if err != nil {
		return errors.Wrapf(err, "put index object: %s", i.index.IndexLocation())
	}
	klog.Infof("backup etcd finished: %s", i.index.Path(backup))
	return nil
}

func (i *Indexer) BackupGC(id string) error {
	i.lock.Lock()
	defer i.lock.Unlock()

	if i.index == nil {
		err := i.LoadIndex(id)
		if err != nil {
			return errors.Wrapf(err, "load index")
		}
	}

	if len(i.index.Copies) <= KEEP_COPIES_CNT {
		return nil
	}
	i.index.SortBackups()
	deleted := false
	var bck []Backup
	for k, backup := range i.index.Copies {
		if k < KEEP_COPIES_CNT {
			bck = append(bck, backup)
			continue
		}
		deleted = true
		err := i.store.DeleteObject(i.index.Path(backup))
		klog.Infof("remove etcd backup copies: %s, %v", i.index.Path(backup), err)
	}
	if deleted {
		i.index.Copies = bck
		err := i.store.PutObject(i.index.Bytes(), i.index.IndexLocation())
		if err != nil {
			klog.Errorf("clean up, put index object fail: %i", err.Error())
		}
	}
	klog.Infof("clean up backups: %d", len(i.index.Copies))
	return nil
}

func NewIndex(name string) *BackupIndex {
	return &BackupIndex{
		Prefix: "ovm/backup",
		Name:   name,
	}
}

const SnapshotTMP = "/tmp/snapshot.db"

type BackupIndex struct {
	Prefix string           `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Name   string           `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	Copies []Backup         `json:"copies,omitempty" protobuf:"bytes,3,opt,name=copies"`
	Spec   *api.ClusterSpec `json:"spec,omitempty" protobuf:"bytes,4,opt,name=spec"`
}

type Backup struct {
	Identity string `json:"identity,omitempty" protobuf:"bytes,1,opt,name=identity"`
}

func (i *BackupIndex) base() string {
	return fmt.Sprintf("%s/%s", i.Prefix, i.Name)
}
func (i *BackupIndex) IndexLocation() string {
	return fmt.Sprintf("%s/index.json", i.base())
}

func (i *BackupIndex) URI(b Backup, bName string) string {
	return fmt.Sprintf("oss://%s/%s", bName, i.Path(b))
}

func (i *BackupIndex) Path(b Backup) string {
	return fmt.Sprintf("%s/%s/snapshot.db", i.base(), b.Identity)
}

func (i *BackupIndex) Load(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, i)
}

func (i *BackupIndex) String() string { return utils.PrettyJson(i) }

func (i *BackupIndex) Bytes() []byte { return []byte(utils.PrettyJson(i)) }

func (i *BackupIndex) SortBackups() {
	cmp := func(m, n int) bool {
		return i.Copies[m].Identity > i.Copies[n].Identity
	}
	sort.Slice(i.Copies, cmp)
}

func (i *BackupIndex) LatestBackup() *Backup {
	i.SortBackups()
	if len(i.Copies) <= 0 {
		return nil
	}
	return &i.Copies[0]
}

func HourNow() string {
	return time.Now().Format("20060102-1504")
}
