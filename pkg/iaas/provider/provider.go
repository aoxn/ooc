package provider

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"k8s.io/klog/v2"
	"sync"
)

type Id struct {
	Name       string
	ResourceId string
	ExtraRIDs  []string
	CreatedAt  string
	UpdatedAt  string
	Options    *v1.OocOptions
}

func (i *Id) String() string {
	return fmt.Sprintf("id://%s/%s/%s", i.ResourceId, i.Name, i.UpdatedAt)
}

func NewContext(
	spec *v1.ClusterSpec,
) *Context {
	return NewOocContext(&v1.OocOptions{}, spec)
}

func NewOocContext(
	cfg *v1.OocOptions,
	spec *v1.ClusterSpec,
) *Context {
	ctx := &Context{}
	ctx.SetKV("BootCFG", spec)
	ctx.SetKV("OocOptions", cfg)
	return ctx
}

type Context struct{ sync.Map }

func (n *Context) BootCFG() *v1.ClusterSpec {
	val, ok := n.Load("BootCFG")
	if !ok {
		klog.Infof("BootCFG not found")
		return &v1.ClusterSpec{}
	}
	return val.(*v1.ClusterSpec)
}

func (n *Context) OocOptions() *v1.OocOptions {
	val, ok := n.Load("OocOptions")
	if !ok {
		klog.Infof("OocOptions not found")
		return &v1.OocOptions{}
	}
	return val.(*v1.OocOptions)
}

func (n *Context) Stack() map[string]Value {
	val, ok := n.Load("Stack")
	if !ok {
		klog.Infof("Stack not found")
		return map[string]Value{}
	}
	return val.(map[string]Value)
}

func (n *Context) WithStack(
	stack map[string]Value,
)*Context {
	n.SetKV("Stack", stack)
	return n
}

func (n *Context) Visit(set func(cache *Context)) { set(n) }

func (n *Context) SetKV(key string, value interface{}) { n.Store(key, value) }

var Providers = sync.Map{}

func AddProvider(key string, value Interface) { Providers.Store(key, value) }

func GetProvider(key string) Interface {

	pvd, ok := Providers.Load(key)
	if !ok {
		panic(fmt.Sprintf("provider %s not supported", key))
	}
	return pvd.(Interface)
}

type Interface interface {
	Resource
	Scaling
	BucketOSS
	NodeOperation
	LocalCache
	NodeGroup
	Initialize(ctx *Context) error
	Create(ctx *Context) (*Id, error)
	WatchResult(ctx *Context, id *Id) error
	Delete(ctx *Context, id *Id) error
}

// Value parameters or outputs for provider interface
// Key specifies the action name
// Val for specific value, could be any structure.
// A common case is json.RawMessage.
// Every provider could interpret it by themselves
type Value struct {
	// Key
	Key 	string
	// Val
	Val   	interface{}
}

type Option struct {
	Action 	string
	Value   Value
}

type BucketOSS interface {
	GetFile(src, dst string) error
	PutFile(src, dst string) error
	DeleteObject(f string) error
	GetObject(src string) ([]byte, error)
	PutObject(b []byte, dst string) error
}

type Resource interface {
	GetStackOutPuts(ctx *Context, id *Id) (map[string]Value, error)
	GetInfraStack(ctx *Context, id *Id) (map[string]Value, error)
}

type Scaling interface {
	// ModifyScalingConfig etc. UserData
	ModifyScalingConfig(ctx *Context, gid string,opt... Option) error

	ScalingGroupDetail(ctx *Context, gid string, opt Option) (ScaleGroupDetail, error)

	ScaleNodeGroup(ctx *Context,gid string, desired int) error

	ScaleMasterGroup(ctx *Context, gid string, desired int) error

	RemoveScalingGroupECS(ctx *Context, gid string, ecs string) error
}

type NodeGroup interface {
	CreateNodeGroup(ctx *Context, np *v1.NodePool) (*v1.BindID, error)

	DeleteNodeGroup(ctx *Context, np *v1.NodePool) error

	ModifyNodeGroup(ctx *Context, np *v1.NodePool) error
}

type NodeOperation interface {

	TagECS(ctx *Context, id string, val... Value) error

	RestartECS(ctx *Context, id string) error

	ReplaceSystemDisk(ctx *Context, id string, opt Option) error
}

type LocalCache interface {
	Load(ctx *Context) ([]*v1.ClusterSpec, error)
	Save(ctx *Context, id *Id, n *v1.ClusterSpec) error
}


type ScaleGroupDetail struct {
	GroupId 	string
	Instances   map[string]Instance
}

type Instance struct {
	Id 	string
	Ip  string

	Tags 	[]Value

	CreatedAt string

	UpdatedAt string

	// Status Stop|Running
	Status 	  string
}