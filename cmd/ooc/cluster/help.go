package cluster

import (
	"fmt"
	v1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider/alibaba"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

type flagpole struct {
	Show string
}

func NewCommandConfig() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Kubernetes config --show context.cfg",
		Long:  "kubernetes show template config",
		RunE: func(cmd *cobra.Command, args []string) error {

			//return test(flags,cmd,args)
			return show(flags)
		},
	}
	cmd.Flags().StringVar(&flags.Show, "show", "context.cfg", "show config template:  context.cfg|kubeconfig")
	return cmd
}

func NewContextCFG() *v1.ContextCFG {
	alibabadev := alibaba.AlibabaDev{
		BucketName:      "ovm-index",
		Region:          "cn-hangzhou",
		AccessKeyId:     "xxxxxxxxxx",
		AccessKeySecret: "YYYYYYYYYYYYYYY",
	}

	raw, err := v1.ToRawMessage(alibabadev)
	if err != nil {
		klog.Errorf("some error occurred: %s", err.Error())
	}
	cfg := v1.ContextCFG{
		Kind:           "Config",
		APIVersion:     v1.SchemeGroupVersion.String(),
		CurrentContext: "devEnv",
		Contexts: []v1.ContextItem{
			{Name: "devEnv", Context: &v1.Context{ProviderKey: "alibaba.dev"}},
		},
		Providers: []v1.ProviderItem{
			{Name: "alibaba.dev", Provider: &v1.Provider{Name: "alibaba", Value: raw}},
		},
	}
	return &cfg
}

func show(flag *flagpole) error {
	switch flag.Show {
	case "kubeconfig":
		fmt.Printf("%s\n", help)
	case "context.cfg":
		fmt.Printf(utils.PrettyYaml(NewContextCFG()))
	}
	return nil
}

var help = `
## --cluster-config config cluster specification
"
clusterid: ${CLUSTER_ID}
iaas:
  workerCount: 3
  image: centos_7_06_64_20G_alibase_20190218.vhd
  disk:
    size: 40G
    type: cloudssd
  zoneid: cn-hangzhou-g
  instance: ecs.c5.xlarge
registry: ${REGISTRY}
namespace: ${NAMESPACE}
cloudType: ${CLOUD_TYPE}
kubernetes:
  name: kubernetes
  version: 1.12.6-aliyun.1
etcd:
  name: etcd
  version: v3.3.8
runtime:
  name: runtime
  version: 18.09.2
  para:
    key1: value
    key2: value2
sans:
  - 192.168.0.1
network:
  mode: ipvs
  podcidr: 172.16.0.1/16
  svccidr: 172.19.0.1/20
  domain: cluster.domain
  netMask: ${NET_MASK}
endpoint:
  intranet: ${INTRANET_LB}
  internet: ${INTERNET_LB}
"
`
