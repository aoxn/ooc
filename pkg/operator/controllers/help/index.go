package help

import (
	"encoding/json"
	"fmt"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	"sort"
)

func NewIndex(name string) *Index{
	return &Index{
		Prefix: "backup",
		Name:   name,
		Bucket: "host-oc-cn-hangzhou",
	}
}

const SnapshotTMP = "/tmp/snapshot.db"

type Index struct {
	Prefix string 	`json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`
	Bucket string   `json:"bucket,omitempty" protobuf:"bytes,2,opt,name=bucket"`
	Name   string   `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	Copies []Backup `json:"copies,omitempty" protobuf:"bytes,4,opt,name=copies"`
	Spec   *api.Cluster `json:"spec,omitempty" protobuf:"bytes,5,opt,name=spec"`
}

type Backup struct {
	Identity string	`json:"identity,omitempty" protobuf:"bytes,1,opt,name=identity"`
}

func (i *Index) IndexLocation() string {
	return fmt.Sprintf("oss://%s/%s/%s/index.json",i.Bucket,i.Prefix,i.Name)
}

func (i *Index) Path(b Backup) string {
	return fmt.Sprintf("oss://%s/%s/%s/%s/snapshot.db", i.Bucket,i.Prefix,i.Name,b.Identity)
}

func(i *Index) LoadIndex(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b,i)
}

func(i *Index) String() string { return utils.PrettyJson(i) }

func (i *Index) Bytes() []byte { return []byte(utils.PrettyJson(i)) }


func (i *Index) SortBackups() {
	cmp := func(m,n int) bool {
		return i.Copies[m].Identity > i.Copies[n].Identity
	}
	sort.Slice(i.Copies, cmp)
}

func (i *Index) LatestBackup() *Backup {
	i.SortBackups()
	if len(i.Copies) <= 0{
		return nil
	}
	return &i.Copies[0]
}