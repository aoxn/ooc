package iaas

import (
	"encoding/base64"
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	_ "github.com/aoxn/ooc/pkg/iaas/provider/alibaba"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/aoxn/ooc/pkg/utils/sign"
	"github.com/pkg/errors"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

func Recover(cfg *v1.OocOptions, name string) error {
	ctx, err := provider.NewContext(cfg, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}

	index := ctx.Indexer()
	id, err := index.Get(name)
	if err != nil {
		return errors.Wrapf(err, "no cluster found by name %s", name)
	}

	ctx.SetKV("BootCFG", &id.Spec.Cluster)
	pvd := ctx.Provider()
	_, err = pvd.Recover(ctx, &id)
	return err
}

func SetDefaultCA(spec *v1.ClusterSpec) {
	if spec.Kubernetes.RootCA == nil {
		root, err := context.NewKeyCert()
		if err != nil {
			panic("new self signed cert pair fail")
		}
		spec.Kubernetes.RootCA = root
	}
}

func Create(cfg *v1.OocOptions) error {
	ctx, err := provider.NewContext(cfg, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}
	bootcfg := ctx.BootCFG()
	// GetProvider will return error if bootcfg.BindInfra.Options.Name is not correct
	pvd := ctx.Provider()
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider: %s", cfg.Default.CurrentContext)
	}

	SetDefaultCA(bootcfg)
	id := v1.ClusterId{
		ObjectMeta: metav1.ObjectMeta{
			Name: bootcfg.ClusterID,
		},
		Spec: v1.ClusterIdSpec{
			Options:   cfg,
			Cluster:   *bootcfg,
			CreatedAt: time.Now().Format("2006-01-02T15:04:05"),
			UpdatedAt: time.Now().Format("2006-01-02T15:04:05"),
		},
	}
	indexer := ctx.Indexer()
	err = indexer.Save(id)
	if err != nil {
		return errors.Wrapf(err, "create cluster: %s", id.Name)
	}

	nid, err := pvd.Create(ctx)
	if err != nil {
		return fmt.Errorf("call provider [%s] create: %s", bootcfg.Bind.Provider.Name, err.Error())
	}
	// set id for defer function.
	err = indexer.Save(*nid)
	if err != nil {
		klog.Errorf("save cluster cache after: %s", err.Error())
	}
	klog.Infof("cluster created: %s", utils.PrettyYaml(id))
	klog.Infof("watch cluster create progress with command:  [ ooc watch --name %s ] ", id.Name)
	return nil
}

func Delete(options *v1.OocOptions, name string) error {
	if name == "" {
		return fmt.Errorf("cluster name must be provided with --name")
	}
	ctx, err := provider.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}

	// GetProvider will return error if bootcfg.BindInfra.Options.Name is not correct
	pvd := ctx.Provider()
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider: %s", options.Default.CurrentContext)
	}

	index := ctx.Indexer()
	id, err := index.Get(name)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			klog.Infof("cluster[%s] not found, finish", name)
			return nil
		}
		return errors.Wrapf(err, "delete cluster: %s", name)
	}
	err = pvd.Delete(ctx, &id)
	if err != nil {
		return fmt.Errorf("delete cluster: %s", err.Error())
	}
	return index.Remove(name)
}

func Scale(options *v1.OocOptions, name string, count int) error {
	ctx, err := provider.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}
	index := ctx.Indexer()
	id, err := index.Get(name)
	if err != nil {
		return errors.Wrapf(err, "scale cluster: %s", name)
	}
	pvd := ctx.Provider()
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider")
	}
	err = pvd.ScaleMasterGroup(ctx, "", count)
	if err != nil {
		return fmt.Errorf("scale cluster: %s", err.Error())
	}
	id.Spec.UpdatedAt = time.Now().Format("2006-01-02T15:04:05")
	return index.Save(id)
}

func Get(options *v1.OocOptions, cmdLine *v1.CommandLineArgs) error {
	switch options.Resource {
	case "backup":
		return doGetBuckups(options, cmdLine)
	case "kubeconfig":
		return doGetKubeConfig(options, cmdLine)
	}
	return doGetCluster(options, cmdLine)
}

func doGetCluster(options *v1.OocOptions, cmdLine *v1.CommandLineArgs) error {
	ctx, err := provider.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}
	index := ctx.Indexer()
	if options.ClusterName == "" {
		ids, err := index.List(options.ClusterName)
		if err != nil {
			return errors.Wrapf(err, "ListCluster")
		}
		switch cmdLine.OutPutFormat {
		case "yaml":
			for _, v := range ids {
				fmt.Printf(utils.PrettyYaml(v.Spec.Cluster))
				fmt.Println("\n\n---")
			}
		case "json":
			for _, v := range ids {
				fmt.Printf(utils.PrettyJson(v.Spec.Cluster))
				fmt.Println("\n\n---")
			}
		default:
			klog.Info()
			fmt.Printf("%-20s%-40s\n", "NAME", "ENDPOINT")
			for i := range ids {
				fmt.Printf("%-20s%-40s\n", ids[i].Name, ids[i].Spec.Cluster.Endpoint.Internet)
			}
		}
	} else {
		id, err := index.Get(options.ClusterName)
		if err != nil {
			return errors.Wrapf(err, "get cluster: %s", options.ClusterName)
		}
		switch cmdLine.OutPutFormat {
		case "yaml":
			fmt.Printf(utils.PrettyYaml(id.Spec.Cluster))
		case "json":
			fmt.Printf(utils.PrettyJson(id.Spec.Cluster))
		default:
			klog.Info()
			fmt.Printf("%-20s%-40s\n", "NAME", "ENDPOINT")
			fmt.Printf("%-20s%-40s\n", id.Name, id.Spec.Cluster.Endpoint.Internet)
		}
	}
	return nil
}

func doGetBuckups(options *v1.OocOptions, cmdLine *v1.CommandLineArgs) error {
	if options.ClusterName == "" {
		return fmt.Errorf("cluster name must be specified over [-n xxx]")
	}
	ctx, err := provider.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}
	index := ctx.Indexer()
	backups, err := index.ListBackups(options.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "get backup files")
	}
	backups.SortBackups()
	switch cmdLine.OutPutFormat {
	case "yaml":
		fmt.Printf(utils.PrettyYaml(backups))
	case "json":
		fmt.Printf(utils.PrettyJson(backups))
	default:
		klog.Info()
		fmt.Printf("%-20s%-20s%-20s%-80s\n", "NAME", "PREFIX", "DATE", "PATH")
		for _, b := range backups.Copies {
			fmt.Printf("%-20s%-20s%-20s%-80s\n", backups.Name, backups.Prefix, b.Identity, backups.Path(b))
		}
	}
	return nil
}

func doGetKubeConfig(options *v1.OocOptions, cmdLine *v1.CommandLineArgs) error {
	if options.ClusterName == "" {
		return fmt.Errorf("cluster name must be specified over [-n xxx]")
	}
	ctx, err := provider.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}
	index := ctx.Indexer()
	id, err := index.Get(options.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "scale cluster: %s", options.ClusterName)
	}
	if id.Spec.Cluster.Kubernetes.RootCA == nil {
		return fmt.Errorf("root ca does not exist in spec.Kubernetes.RootCA in id cache")
	}
	key, crt, err := sign.SignKubernetes(
		id.Spec.Cluster.Kubernetes.RootCA.Cert, id.Spec.Cluster.Kubernetes.RootCA.Key, []string{},
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
			AuthCA:    base64.StdEncoding.EncodeToString(id.Spec.Cluster.Kubernetes.RootCA.Cert),
			Address:   id.Spec.Cluster.Endpoint.Internet,
			ClientCRT: base64.StdEncoding.EncodeToString(crt),
			ClientKey: base64.StdEncoding.EncodeToString(key),
		},
	)
	if err != nil {
		return fmt.Errorf("render admin.local config error: %s", err.Error())
	}

	if cmdLine.WriteTo == "" {
		fmt.Printf(cfg)
		return nil
	} else {
		//mpath := filepath.Join(os.Getenv("HOME"), ".kube/config.ooc")
		mpath := cmdLine.WriteTo
		err = ioutil.WriteFile(mpath, []byte(cfg), 0755)
		if err == nil {
			klog.Infof("write kubeconfig to file [%s]", mpath)
		} else {
			klog.Errorf("write kubeconfig to %s failed: %s", mpath, err.Error())
		}
	}
	return nil
}

func WatchResult(options *v1.OocOptions, name string) error {
	ctx, err := provider.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize ooc context")
	}
	index := ctx.Indexer()
	id, err := index.Get(name)
	if err != nil {
		return errors.Wrapf(err, "scale cluster: %s", name)
	}

	pvd := ctx.Provider()
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider")
	}
	err = pvd.WatchResult(ctx, &id)
	if err != nil {
		return fmt.Errorf("watch error: %s", err.Error())
	}
	return index.Save(id)
}
