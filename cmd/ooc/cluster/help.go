package cluster

import (
	"fmt"
	"github.com/spf13/cobra"
)

type flagpole struct {
	Show bool
	Kubeconfig bool
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommandConfig() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Kubernetes config --show-tpl",
		Long:  "kubernetes show tpl config",
		RunE: func(cmd *cobra.Command, args []string) error {

			//return test(flags,cmd,args)
			return show(flags)
		},
	}
	cmd.Flags().BoolVar(&flags.Show, "show-tpl", true, "show config template")
	cmd.Flags().BoolVar(&flags.Kubeconfig, "show-kubeconfig", true, "show kubeconfig")
	return cmd
}



func show(flag *flagpole) error {
	fmt.Printf("%s\n", help)
	return nil
}

var help = `
## --provider-config-file for ROS provider
"
accessKey: abcdefg.hsxx
accessKeySecret: amkkdiillllddkmmmmmmmm
templateFile: /Users/aoxn/work/ooc/pkg/iaas/provider/ros/demo.dev.json
"

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
