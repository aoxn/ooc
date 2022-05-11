package provider

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/aoxn/wdrip/pkg/utils/cmd"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"sync"
)

func NewContext(
	options *v1.WdripOptions, spec *v1.ClusterSpec,
) (*Context, error) {
	mctx := &Context{}
	mctx.SetKV("BootCFG", spec)
	mctx.SetKV("WdripOptions", options)
	return mctx, mctx.Initialize(options)
}

func NewEmptyContext() *Context { return &Context{} }

func NewContextWithCluster(spec *v1.ClusterSpec) *Context {
	mctx := &Context{}
	mctx.SetKV("BootCFG", spec)
	return mctx
}

type Context struct{ sync.Map }

func (n *Context) Initialize(opts *v1.WdripOptions) error {
	n.SetKV("WdripOptions", opts)
	if opts.Default == nil {
		opts.Default = BuildContexCFG(n.BootCFG())
	}
	dprvd := opts.Default.CurrentPrvdCFG()
	if opts.Config != "" {
		bootcfg, err := LoadBootCFG(opts.Config)
		if err != nil {
			return fmt.Errorf("load bootcfg: %s", err.Error())
		}
		if bootcfg.Bind.Provider == nil {
			bootcfg.Bind.Provider = dprvd
		}
		// cluster config provider is in higher priority
		dprvd = bootcfg.Bind.Provider
		n.SetKV("BootCFG", bootcfg)
		klog.Infof("use command line config "+
			"as bootconfig: [%s] with provider[%s]", opts.Config, dprvd.Name)
	}

	// GetProvider will return error if
	// bootcfg.BindInfra.Options.Name is not correct
	pvd := GetProvider(dprvd.Name)
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider: %s", dprvd.Name)
	}
	err := pvd.Initialize(n)
	if err != nil {
		return fmt.Errorf("initialize provider: %s", err.Error())
	}

	n.SetKV("Provider", pvd)
	return nil
}

func (n *Context) Provider() Interface {
	val, ok := n.Load("Provider")
	if !ok {
		klog.Infof("Provider not found")
		return val.(Interface)
	}
	return val.(Interface)
}

func (n *Context) BootCFG() *v1.ClusterSpec {
	val, ok := n.Load("BootCFG")
	if !ok {
		klog.Infof("BootCFG not found")
		return &v1.ClusterSpec{}
	}
	return val.(*v1.ClusterSpec)
}

func (n *Context) WdripOptions() *v1.WdripOptions {
	val, ok := n.Load("WdripOptions")
	if !ok {
		klog.Infof("WdripOptions not found")
		return &v1.WdripOptions{}
	}
	return val.(*v1.WdripOptions)
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
) *Context {
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

const (
	MasterUserdata     = "Master"
	WorkerUserdata     = "Worker"
	RecoverUserdata    = "Recover"
	JoinMasterUserdata = "JoinMaster"
)

type Interface interface {
	Resource
	Scaling
	ObjectStorage
	NodeOperation
	NodeGroup
	UserData(ctx *Context, category string) (string, error)
	Initialize(ctx *Context) error
	Create(ctx *Context) (*v1.ClusterId, error)
	Recover(ctx *Context, id *v1.ClusterId) (*v1.ClusterId, error)
	WatchResult(ctx *Context, id *v1.ClusterId) error
	Delete(ctx *Context, id *v1.ClusterId) error
}

// Value parameters or outputs for provider interface
// Key specifies the action name
// Val for specific value, could be any structure.
// A common case is json.RawMessage.
// Every provider could interpret it by themselves
type Value struct {
	// Key
	Key string
	// Val
	Val interface{}
}

type Option struct {
	Action string
	Value  Value
}

type ObjectStorage interface {
	BucketName() string
	EnsureBucket(name string) error
	GetFile(src, dst string) error
	PutFile(src, dst string) error
	DeleteObject(f string) error
	GetObject(src string) ([]byte, error)
	PutObject(b []byte, dst string) error
	ListObject(prefix string) ([][]byte, error)
}

type Resource interface {
	GetStackOutPuts(ctx *Context, id *v1.ClusterId) (map[string]Value, error)
	GetInfraStack(ctx *Context, id *v1.ClusterId) (map[string]Value, error)
}

type Scaling interface {
	VSwitchs(ctx *Context) (string, error)

	// ModifyScalingConfig etc. UserData
	ModifyScalingConfig(ctx *Context, gid string, opt ...Option) error

	ScalingGroupDetail(ctx *Context, gid string, opt Option) (ScaleGroupDetail, error)

	ScaleNodeGroup(ctx *Context, gid string, desired int) error

	ScaleMasterGroup(ctx *Context, gid string, desired int) error

	RemoveScalingGroupECS(ctx *Context, gid string, ecs string) error
}

type NodeGroup interface {
	CreateNodeGroup(ctx *Context, np *v1.NodePool) (*v1.BindID, error)

	DeleteNodeGroup(ctx *Context, np *v1.NodePool) error

	ModifyNodeGroup(ctx *Context, np *v1.NodePool) error
}

type NodeOperation interface {
	TagECS(ctx *Context, id string, val ...Value) error

	InstanceDetail(ctx *Context, id []string) ([]Instance, error)

	StopECS(ctx *Context, id string) error

	DeleteECS(ctx *Context, id string) error

	RestartECS(ctx *Context, id string) error

	ReplaceSystemDisk(ctx *Context, id string, userdata string, opt Option) error

	RunCommand(ctx *Context, id, cmd string) (Result, error)
}

type Result struct {
	Status string
	OutPut string
}

type ScaleGroupDetail struct {
	GroupId   string
	Instances map[string]Instance
}

type Instance struct {
	Region string
	Id     string
	Ip     string

	Tags []Value

	CreatedAt string

	UpdatedAt string

	// Status Stop|Running
	Status string

	GetNodeName func() string
}

func LoadBootCFG(name string) (*v1.ClusterSpec, error) {
	if name == "" {
		return nil, fmt.Errorf("cluster config file must be specified with --config")
	}
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read boot config file: %s", err.Error())
	}
	spec := &v1.ClusterSpec{}
	if err := yaml.Unmarshal(data, spec); err != nil {
		return nil, fmt.Errorf("unmarshal bootcfg: %s", err.Error())
	}
	return spec, nil
}

func BuildContexCFG(spec *v1.ClusterSpec) *v1.ContextCFG {
	home, err := HomeDir()
	if err != nil {
		klog.Warningf("failed to find HOME dir by $(pwd ~)")
	}
	klog.Infof("use HOME dir: [%s]", home)
	cacheDir := filepath.Join(home, ".wdrip/")
	/*
		sequence:
		1. from local wdrip config, ~/.wdrip/config
		2. from cluster spec provider
	*/
	mctx := v1.ContextCFG{
		Kind:       "Config",
		APIVersion: v1.SchemeGroupVersion.String(),
	}
	mfi := filepath.Join(cacheDir, "config")
	exist, err := utils.FileExist(mfi)
	if err == nil {
		if exist {
			klog.Infof("trying to load context config from: %s", mfi)
			cfg, err := ioutil.ReadFile(mfi)
			if err != nil {
				klog.Warningf("read wdrip default config: %s", err.Error())
			}
			err = yaml.Unmarshal(cfg, &mctx)
			if err != nil {
				klog.Warningf("unmarshal config: %s", err.Error())
			}
			if mctx.CurrentContext == "" || len(mctx.Contexts) == 0 {
				klog.Warningf("no current context "+
					"or providers section: %s", mctx.CurrentContext, len(mctx.Contexts))
			}
		} else {
			klog.Infof("wdrip config[%s] does not exist", mfi)
		}
	}
	if spec != nil {
		klog.Infof("cluster spec not empty")
		if spec.Bind.Provider != nil {
			pkey := "provider01"
			mctx.Contexts = []v1.ContextItem{
				{Name: spec.ClusterID, Context: &v1.Context{ProviderKey: pkey}},
			}

			mctx.Providers = []v1.ProviderItem{
				{Name: pkey, Provider: spec.Bind.Provider},
			}
			mctx.CurrentContext = spec.ClusterID
			klog.Infof("build context config from cluster spec")
		} else {
			klog.Errorf("cluster spec provider not defined, failed to load provider information")
		}
	}
	if mctx.CurrentContext == "" {
		klog.Warningf("empty provider config, system would not work")
	}
	return &mctx
}

func HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil {
		return home, nil
	}
	// cloud-init does not have a HOME env on startup
	// workaround by using $(whoami)
	klog.Errorf("home dir not found "+
		"in $HOME: [%s] try command $(whoami)", err.Error())
	cm := cmd.NewCmd("whoami")
	result := <-cm.Start()
	err = cmd.CmdError(result)
	if err != nil {
		return "", errors.Wrapf(err, "read home dir by $(pwd ~)")
	}
	if len(result.Stdout) <= 0 {
		klog.Warningf("$(whoami) has no stdand output")
		return "", nil
	}
	who := result.Stdout[0]
	if who == "root" {
		return "/root", nil
	}
	return fmt.Sprintf("/home/%s", result.Stdout[0]), nil
}
