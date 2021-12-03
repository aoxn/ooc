package iaas

import (
	"encoding/base64"
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	_ "github.com/aoxn/ooc/pkg/iaas/provider/dev"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/aoxn/ooc/pkg/utils/sign"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"time"
)

var CachDir = filepath.Join(os.Getenv("HOME"), ".ooc/")

func Create(cfg *v1.OocOptions) error {
	err := os.MkdirAll(CachDir, 0755)
	if err != nil {
		return fmt.Errorf("make cache dir: %s", err.Error())
	}
	bootcfg, err := LoadBootCFG(cfg.Config)
	if err != nil {
		return fmt.Errorf("load bootcfg: %s", err.Error())
	}
	ctx := provider.NewOocContext(cfg, bootcfg)

	// GetProvider will return error if bootcfg.BindInfra.Options.Name is not correct
	pvd := provider.GetProvider(bootcfg.Bind.Provider.Name)
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider: %s", bootcfg.Bind.Provider.Name)
	}

	utils.SetDefaultCA(bootcfg)
	id := provider.Id{
		Options:    cfg,
		Name:       bootcfg.ClusterID,
		CreatedAt:  time.Now().Format("2006-01-02T15:04:05"),
		UpdatedAt:  time.Now().Format("2006-01-02T15:04:05"),
	}
	err = pvd.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("initialize provider: %s", err.Error())
	}
	defer func(
		spvd provider.Interface,
		sid *provider.Id,
		sctx *provider.Context,
		sbootcfg *v1.ClusterSpec,
	) {
		klog.Infof("resource: %+v", sid)
		err = SaveCache(spvd, sid, sctx, sbootcfg)
		if err != nil {
			klog.Errorf("save cluster cache: %s", err.Error())
		}
	}(pvd, &id, ctx, bootcfg)
	if err != nil {
		return fmt.Errorf("create initialize provider: %s", err.Error())
	}
	nid, err := pvd.Create(ctx)
	if err != nil {
		return fmt.Errorf("call provider [%s] create: %s", bootcfg.Bind.Provider.Name, err.Error())
	}
	// set id for defer function.
	id = *nid
	err = SaveCache(pvd, &id, ctx, bootcfg)
	if err != nil {
		klog.Errorf("save cluster cache after: %s", err.Error())
	}
	klog.Infof("cluster created: %s", utils.PrettyYaml(id))
	klog.Infof("watch cluster create progress with command:  [ ooc watch --name %s ] ",id.Name)
	return nil
}

func Delete(name string, region string) error {
	id, spec, err := LoadCache(name)
	if err != nil {
		if ne,ok := err.(*NotExist);ok {
			klog.Infof("cluster[%s] not found, finish, %s", name,ne)
			return nil
		}
		return fmt.Errorf("load stack id: %s", err.Error())
	}
	pvd := provider.GetProvider(spec.Bind.Provider.Name)
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider")
	}
	ctx := provider.NewOocContext(id.Options, spec)
	err = pvd.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("delete initialize provider: %s", err.Error())
	}
	err = pvd.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete cluster: %s", err.Error())
	}
	return RemoveCache(name)
}

func Scale(name string, count int) error {
	id, spec, err := LoadCache(name)
	if err != nil {
		return fmt.Errorf("load stack id: %s", err.Error())
	}
	pvd := provider.GetProvider(spec.Bind.Provider.Name)
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider")
	}
	ctx := provider.NewOocContext(id.Options, spec)
	err = pvd.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("scale initialize provider: %s", err.Error())
	}
	err = pvd.ScaleMasterGroup(ctx, "", count)
	if err != nil {
		return fmt.Errorf("scale cluster: %s", err.Error())
	}
	id = &provider.Id{
		Options:    ctx.OocOptions(),
		ResourceId: id.ResourceId,
		Name:       id.Name,
		CreatedAt:  id.CreatedAt,
		UpdatedAt:  time.Now().Format("2006-01-02T15:04:05"),
	}
	return SaveCache(pvd, id, ctx, spec)
}

func Get(name string) error {
	info, err := ioutil.ReadDir(CachDir)
	if err != nil {
		return fmt.Errorf("read cluster: %s", err.Error())
	}
	if name == "" {
		fmt.Printf("%-20s\n", "NAME")
		for i := range info {
			if info[i].IsDir() {
				continue
			}
			fmt.Printf("%-20s\n", info[i].Name())
		}
	} else {
		data, err := ioutil.ReadFile(filepath.Join(CachDir, name))
		if err != nil {
			return fmt.Errorf("read cluster %s: %s", name, err.Error())
		}
		fmt.Println(string(data))
	}
	return nil
}

func KubeConfig(name string) error {
	_, spec, err := LoadCache(name)
	if err != nil {
		if ne,ok := err.(*NotExist);ok {
			klog.Infof("cluster[%s] not found, %s", name,ne)
			return nil
		}
		return fmt.Errorf("load stack id: %s", err.Error())
	}
	if spec.Kubernetes.RootCA == nil {
		return fmt.Errorf("root ca does not exist in spec.Kubernetes.RootCA in id cache")
	}
	key,crt, err := sign.SignKubernetes(
		spec.Kubernetes.RootCA.Cert, spec.Kubernetes.RootCA.Key,[]string{},
	)
	if err != nil {
		return fmt.Errorf("sign kubernetes crt: %s", err.Error())
	}
	cfg, err := utils.RenderConfig(
		"admin.cfg",
		utils.KubeConfigTpl,
		struct {
			AuthCA    string
			Address   string
			ClientCRT string
			ClientKey string
		}{
			AuthCA:    base64.StdEncoding.EncodeToString(spec.Kubernetes.RootCA.Cert),
			Address:   spec.Endpoint.Internet,
			ClientCRT: base64.StdEncoding.EncodeToString(crt),
			ClientKey: base64.StdEncoding.EncodeToString(key),
		},
	)
	if err != nil {
		return fmt.Errorf("render admin.local config error: %s", err.Error())
	}
	fmt.Printf(cfg)
	return nil
}

func WatchResult(name string) error {

	id, spec, err := LoadCache(name)
	if err != nil {
		return fmt.Errorf("load stack id: %s", err.Error())
	}
	pvd := provider.GetProvider(spec.Bind.Provider.Name)
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider")
	}
	ctx := provider.NewOocContext(id.Options, spec)
	err = pvd.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("watch initialize provider: %s", err.Error())
	}
	err = pvd.WatchResult(ctx, id)
	if err != nil {
		return fmt.Errorf("watch error: %s", err.Error())
	}
	return SaveCache(pvd,id,ctx, spec)
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

func SaveCache(
	pvd provider.Interface,
	id *provider.Id,
	ctx *provider.Context,
	bootcfg *v1.ClusterSpec,
) error {
	id.Options.Region = bootcfg.Bind.Region
	data, err := yaml.Marshal(
		Cache{
			Id:      id,
			BootCfg: bootcfg,
		},
	)
	if err != nil {
		return fmt.Errorf("save cache marshal: %s", err.Error())
	}
	err = ioutil.WriteFile(
		filepath.Join(CachDir, id.Name),
		data, 0755,
	)
	if err != nil {
		return fmt.Errorf("save cluster cache: %s", err.Error())
	}
	return pvd.Save(ctx, id, bootcfg)
}

func LoadCache(
	name string,
) (*provider.Id, *v1.ClusterSpec, error) {
	id := &Cache{}
	cachefile := filepath.Join(CachDir, name)
	exist, err := utils.FileExist(cachefile)
	if err != nil {
		return id.Id, id.BootCfg, fmt.Errorf("read cluster record: %s", err.Error())
	}
	if !exist {
		return id.Id, id.BootCfg, &NotExist{"NotExist"}
	}

	data, err := ioutil.ReadFile(cachefile)
	if err != nil {
		return nil, nil, fmt.Errorf("load cache: %s", err.Error())
	}
	if err := yaml.Unmarshal(data, id); err != nil {
		return nil, nil, fmt.Errorf("unmarshal id: %s", err.Error())
	}
	return id.Id, id.BootCfg, nil
}

func RemoveCache(
	name string,
) error {
	err := os.Remove(filepath.Join(CachDir, name))
	if err != nil {
		return fmt.Errorf("load cache: %s", err.Error())
	}
	return nil
}

type Cache struct {
	Id      *provider.Id    `json:"id,omitempty" protobuf:"bytes,1,opt,name=id"`
	BootCfg *v1.ClusterSpec `json:"bootcfg,omitempty" protobuf:"bytes,2,opt,name=bootcfg"`
}

type NotExist struct {
	Reason  string
}

func (e *NotExist) Error() string { return e.Reason }
