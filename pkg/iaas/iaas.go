package iaas

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/context"
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
	_ "github.com/aoxn/wdrip/pkg/iaas/provider/alibaba"
	"github.com/aoxn/wdrip/pkg/index"
	h "github.com/aoxn/wdrip/pkg/operator/controllers/help"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/aoxn/wdrip/pkg/utils/sign"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Recover(cfg *v1.WdripOptions) error {
	ctx, err := pd.NewContext(cfg, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}
	idx := index.NewGenericIndexer(cfg.ClusterName, ctx.Provider())
	id, err := idx.GetCluster(cfg.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "no cluster found by name %s", cfg.ClusterName)
	}
	if cfg.RecoverFrom != cfg.ClusterName {
		// recover from another cluster
		mindex := index.NewGenericIndexer(cfg.RecoverFrom, ctx.Provider())
		from, err := mindex.GetCluster(cfg.RecoverFrom)
		if err != nil {
			return errors.Wrapf(err, "no cluster found by name %s", cfg.ClusterName)
		}

		from.Spec.Cluster.Endpoint = id.Spec.Cluster.Endpoint
		from.Spec.Cluster.ClusterID = id.Spec.Cluster.ClusterID
		from.Spec.Cluster.Bind.Provider = id.Spec.Cluster.Bind.Provider
		from.Spec.Cluster.Bind.Region = id.Spec.Cluster.Bind.Region
		from.Spec.Cluster.Bind.ResourceId = id.Spec.Cluster.Bind.ResourceId
		// set back
		id.Spec.Cluster = from.Spec.Cluster
	}
	ctx.SetKV("BootCFG", &id.Spec.Cluster)
	ctx.SetKV("WdripOptions", cfg)
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

func Create(cfg *v1.WdripOptions) error {
	ctx, err := pd.NewContext(cfg, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
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
	indexer := index.NewGenericIndexer(bootcfg.ClusterID, ctx.Provider())
	_, err = indexer.GetCluster(bootcfg.ClusterID)
	if err == nil {
		klog.Warningf("cluster [%s] already exists", bootcfg.ClusterID)
		return errors.Wrapf(err, "cluster [%s] already exists", bootcfg.ClusterID)
	}
	if !strings.Contains(err.Error(), "NoSuchKey") {
		return errors.Wrapf(err, "create cluster")
	}
	err = indexer.SaveCluster(id)
	if err != nil {
		return errors.Wrapf(err, "create cluster: %s", id.Name)
	}

	nid, err := pvd.Create(ctx)
	if err != nil {
		return fmt.Errorf("call provider [%s] create: %s", bootcfg.Bind.Provider.Name, err.Error())
	}
	// set id for defer function.
	err = indexer.SaveCluster(*nid)
	if err != nil {
		klog.Errorf("save cluster cache after: %s", err.Error())
	}
	klog.Infof("cluster created: %s", utils.PrettyYaml(id))
	klog.Infof("watch cluster create progress with command:  [ wdrip watch --name %s ] ", bootcfg.ClusterID)
	return nil
}

func Delete(options *v1.WdripOptions, cmdLine *v1.CommandLineArgs) error {
	if options.ClusterName == "" {
		return fmt.Errorf("cluster name must be provided with --name")
	}
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}

	// GetProvider will return error if bootcfg.BindInfra.Options.Name is not correct
	pvd := ctx.Provider()
	if pvd == nil {
		return fmt.Errorf("unexpected nil provider: %s", options.Default.CurrentContext)
	}

	idx := index.NewGenericIndexer(options.ClusterName, ctx.Provider())
	id, err := idx.GetCluster(options.ClusterName)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			klog.Infof("cluster[%s] not found, finish", options.ClusterName)
			return nil
		}
		return errors.Wrapf(err, "delete cluster: %s", options.ClusterName)
	}
	nidx := index.NewNodePoolIndex(options.ClusterName, ctx.Provider())
	nodepools, err := nidx.ListNodePools("")
	if err != nil {
		return errors.Wrapf(err, "get nodepool oss backups")
	}
	for _, np := range nodepools {
		nodepool := np
		klog.Infof("trying to delete nodepol [%s]", np.Name)
		if err := pvd.DeleteNodeGroup(ctx, &nodepool); err != nil {
			return errors.Wrapf(err, "delete nodegroup %s failed", np.Name)
		}
	}
	err = pvd.Delete(ctx, &id)
	if err != nil {
		if cmdLine.ForceDelete {
			klog.Infof("force delete: %s, %s, %s", options.ClusterName, id.Spec.ResourceId, err.Error())
			return idx.RemoveCluster(options.ClusterName)
		}
		return fmt.Errorf("delete cluster: %s", err.Error())
	}
	return idx.RemoveCluster(options.ClusterName)
}

func Scale(options *v1.WdripOptions, name string, count int) error {
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}
	idx := index.NewGenericIndexer(name, ctx.Provider())
	id, err := idx.GetCluster(name)
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
	return idx.SaveCluster(id)
}

func RunCommand(options *v1.WdripOptions, cmdline *v1.CommandLineArgs) error {
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}

	if cmdline.InstanceID == "" || cmdline.Command == "" {
		return errors.Wrapf(err, "empty instance id or command")
	}
	result, err := ctx.Provider().RunCommand(ctx, cmdline.InstanceID, cmdline.Command)
	if err != nil {
		return errors.Wrapf(err, "run command fail: %s", err.Error())
	}
	klog.Infof("run command on instance [%s] [status=%s]", cmdline.InstanceID, result.Status)
	klog.Infof("\n\n%s", result.OutPut)
	return nil
}

func NodePoolOeration(options *v1.WdripOptions, cmdline *v1.CommandLineArgs) error {
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}
	if options.ClusterName == "" {
		return fmt.Errorf("empty cluster name")
	}
	cidx := index.NewClusterIndex(options.ClusterName, ctx.Provider())
	spec, err := cidx.GetCluster(options.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "get cluster from oss backup")
	}
	stack, err := h.LoadStackFromSpec(ctx.Provider(), ctx, &spec.Spec.Cluster)
	if err != nil {
		return errors.Wrapf(err, "load stack")
	}
	ctx.WithStack(stack)

	fmt.Printf("\n")
	if cmdline.NodePoolID == "" {
		idx := index.NewNodePoolIndex(options.ClusterName, ctx.Provider())
		nodepools, err := idx.ListNodePools("")
		if err != nil {
			return errors.Wrapf(err, "list nodepool from oss backup")
		}
		fmt.Printf("%-20s%-40s%-40s%-40s\n", "NAME", "NODEPOOL_ID", "ESS_ID", "VSWITCH_IDS")
		for _, np := range nodepools {
			bind := np.Spec.Infra.Bind
			if bind == nil {
				bind = &v1.BindID{}
			}
			fmt.Printf("%-20s%-40s%-40s%-40s\n", np.Name, np.Spec.NodePoolID, bind.ScalingGroupId, bind.VswitchIDS)
		}
	} else {
		idx := index.NewNodePoolIndex(options.ClusterName, ctx.Provider())
		np, err := idx.GetNodePool(cmdline.NodePoolID)
		if err != nil {
			return errors.Wrapf(err, "get nodepool from oss backup")
		}
		fmt.Printf("%-20s%-40s%-40s%-20s%-20s\n", "NAME", "NODEPOOL_ID", "INSTANCE_ID", "IP", "STATUS")
		bind := np.Spec.Infra.Bind
		if bind == nil {
			fmt.Printf("%-20s%-40s%-40s%-20s%-20s\n", np.Name, np.Spec.NodePoolID, "", "", "")
			return nil
		}
		detail, err := ctx.Provider().ScalingGroupDetail(ctx, bind.ScalingGroupId, pd.Option{Action: "InstanceIDS"})
		for _, d := range detail.Instances {
			fmt.Printf("%-20s%-40s%-40s%-20s%-20s\n", np.Name, np.Spec.NodePoolID, d.Id, d.Ip, d.Status)
		}
	}
	return nil
}

func Get(options *v1.WdripOptions, cmdLine *v1.CommandLineArgs) error {
	switch options.Resource {
	case "backup":
		return doGetBuckups(options, cmdLine)
	case "kubeconfig":
		return doGetKubeConfig(options, cmdLine)
	}
	return doGetCluster(options, cmdLine)
}

func Edit(options *v1.WdripOptions, cmdLine *v1.CommandLineArgs) error {

	if options.ClusterName == "" {
		return fmt.Errorf("unexpected empty cluster name, specify with --name")
	}
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "edit: initialize wdrip context")
	}
	idx := index.NewGenericIndexer(options.ClusterName, ctx.Provider())
	id, err := idx.GetCluster(options.ClusterName)
	if err != nil {
		return errors.Wrapf(err, "find cluster by name %s", options.ClusterName)
	}
	buf := bytes.NewBufferString(utils.PrettyYaml(id.Spec.Cluster))
	edit := editor.NewDefaultEditor([]string{"EDITOR"})
	edited, _, err := edit.LaunchTempFile(fmt.Sprintf("%s-edit-", filepath.Base(os.Args[0])), "cspec", buf)
	if err != nil {
		return errors.Wrapf(err, "edit with local editor")
	}
	cspec := &v1.ClusterSpec{}
	err = yaml.Unmarshal(edited, cspec)
	if err != nil {
		return errors.Wrapf(err, "unrecognized field or value")
	}
	id.Spec.Cluster = *cspec
	return idx.SaveCluster(id)
}

func doGetCluster(options *v1.WdripOptions, cmdLine *v1.CommandLineArgs) error {
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "get: initialize wdrip context")
	}
	index := index.NewGenericIndexer(options.ClusterName, ctx.Provider())

	if options.ClusterName == "" {
		ids, err := index.ListCluster(options.ClusterName)
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
			fmt.Printf("%-30s%-40s\n", "NAME", "ENDPOINT")
			for i := range ids {
				endp := fmt.Sprintf("%s/%s",
					ids[i].Spec.Cluster.Endpoint.Internet,
					ids[i].Spec.Cluster.Endpoint.Intranet,
				)

				fmt.Printf("%-30s%-40s\n", ids[i].Name, endp)
			}
		}
	} else {
		id, err := index.GetCluster(options.ClusterName)
		if err != nil {
			return errors.Wrapf(err, "get cluster: %s", options.ClusterName)
		}
		endp := fmt.Sprintf("%s/%s",
			id.Spec.Cluster.Endpoint.Internet,
			id.Spec.Cluster.Endpoint.Intranet,
		)

		switch cmdLine.OutPutFormat {
		case "yaml":
			fmt.Printf(utils.PrettyYaml(id.Spec.Cluster))
		case "json":
			fmt.Printf(utils.PrettyJson(id))
		default:
			klog.Info()
			fmt.Printf("%-30s%-40s\n", "NAME", "ENDPOINT")
			fmt.Printf("%-30s%-40s\n", id.Name, endp)
		}
	}
	return nil
}

func doGetBuckups(options *v1.WdripOptions, cmdLine *v1.CommandLineArgs) error {
	if options.ClusterName == "" {
		return fmt.Errorf("cluster name must be specified over [-n xxx]")
	}
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}
	index := index.NewGenericIndexer(options.ClusterName, ctx.Provider())
	backups, err := index.Snapshot()
	if err != nil {
		return errors.Wrapf(err, "backup: get snapshot")
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

func doGetKubeConfig(options *v1.WdripOptions, cmdLine *v1.CommandLineArgs) error {
	if options.ClusterName == "" {
		return fmt.Errorf("cluster name must be specified over [-n xxx]")
	}
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}
	index := index.NewGenericIndexer(options.ClusterName, ctx.Provider())
	id, err := index.GetCluster(options.ClusterName)
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
		//mpath := filepath.Join(os.Getenv("HOME"), ".kube/config.wdrip")
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

func WatchResult(options *v1.WdripOptions, name string) error {
	ctx, err := pd.NewContext(options, nil)
	if err != nil {
		return errors.Wrapf(err, "initialize wdrip context")
	}
	idx := index.NewGenericIndexer(options.ClusterName, ctx.Provider())
	id, err := idx.GetCluster(name)
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
	return idx.SaveCluster(id)
}
