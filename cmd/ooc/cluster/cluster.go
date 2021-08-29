package cluster

import (
	"fmt"
	"github.com/aoxn/ooc/cmd/ooc/version"
	v1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas"
	"github.com/spf13/cobra"
)

const HelpLong = `
## Create a kubernetes cluster with ROS provider
ooc create \
	--name ooc-stack-027 \ 
	--cluster-config /Users/aoxn/work/ooc/pkg/iaas/provider/ros/example/bootcfg.yaml

## Get cluster list
ooc get --resource cluster
or ooc get

## Get cluster specification
ooc get \
	--name ooc-stack-027

## Watch the cluster creation process
ooc watch \
	--name ooc-stack-027

## Delete cluster created by ooc with ROS provider
ooc delete \
	--name ooc-stack-027
`

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &v1.OocOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Kubernetes create cluster",
		Long:  HelpLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf(version.Logo)
			//return test(flags,cmd,args)
			return create(flags)
		},
	}
	cmd.Flags().StringVar(&flags.Resource, "resource", "cluster", "resource eg. cluster")
	//cmd.OocFlags().StringVar(&flags.ProviderCfgFile, "provider-config-file", "", "provider config file")
	cmd.Flags().StringVar(&flags.Config, "config", "", "cluster boot config")
	//cmd.OocFlags().StringVar(&flags.Options, "provider", "ros", "cluster name, support ros")
	cmd.Flags().StringVar(&flags.ClusterName, "name", "", "cluster name")
	return cmd
}

func NewCommandDelete() *cobra.Command {
	flags := &v1.OocOptions{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Kubernetes delete --resource cluster --name clusterid --provider ros",
		Long:  "kubernetes delete cluster. configuration management, lifecycle management",
		RunE: func(cmd *cobra.Command, args []string) error {
			//return test(flags,cmd,args)
			return delete(flags)
		},
	}
	cmd.Flags().StringVar(&flags.Resource, "resource", "cluster", "resource eg. cluster")
	cmd.Flags().StringVar(&flags.ClusterName, "name", "", "cluster name")
	return cmd
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommandScale() *cobra.Command {
	flags := &v1.OocOptions{}
	cmd := &cobra.Command{
		Use:   "scale",
		Short: "Kubernetes scale --resource cluster --name clusterid --target-count 3",
		Long:  "kubernetes scale cluster. configuration management, lifecycle management",
		RunE: func(cmd *cobra.Command, args []string) error {
			//return test(flags,cmd,args)
			return scale(flags)
		},
	}
	cmd.Flags().StringVar(&flags.Resource, "resource", "cluster", "resource eg. cluster")
	cmd.Flags().StringVar(&flags.ClusterName, "name", "", "cluster name")
	cmd.Flags().IntVar(&flags.TargetCount, "target-count", 3, "scale to target count , default to 3 ")
	return cmd
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommandWatch() *cobra.Command {
	flags := &v1.OocOptions{}
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Kubernetes watch --resource cluster --name clusterid ",
		Long:  "kubernetes watch cluster. configuration management, lifecycle management",
		RunE: func(cmd *cobra.Command, args []string) error {
			//return test(flags,cmd,args)
			return watch(flags)
		},
	}
	cmd.Flags().StringVar(&flags.Resource, "resource", "cluster", "resource eg. cluster")
	cmd.Flags().StringVar(&flags.ClusterName, "name", "", "cluster name")
	return cmd
}

var cmdLine = v1.CommandLineArgs{}

func NewCommandGet() *cobra.Command {
	flags := &v1.OocOptions{}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Kubernetes get -r cluster -n clusterid ",
		Long:  "kubernetes get cluster information. ",
		RunE: func(cmd *cobra.Command, args []string) error {
			//return test(flags,cmd,args)
			return get(flags)
		},
	}
	cmd.Flags().StringVarP(&flags.Resource, "resource", "r", "cluster", "resource eg. [cluster|kubeconfig|backup]")
	cmd.Flags().StringVarP(&flags.ClusterName, "name", "n", "", "cluster name")
	cmd.Flags().StringVarP(&cmdLine.WriteTo, "write-to", "w", "", "write config file to the specified destination")
	cmd.Flags().StringVarP(&cmdLine.OutPutFormat, "output", "o", "", "output format [yaml|json]")
	return cmd
}

func get(flags *v1.OocOptions) error    { return iaas.Get(flags, &cmdLine) }
func create(flags *v1.OocOptions) error { return iaas.Create(flags) }
func delete(flags *v1.OocOptions) error { return iaas.Delete(flags, flags.ClusterName) }
func scale(flags *v1.OocOptions) error {
	return iaas.Scale(flags, flags.ClusterName, flags.TargetCount)
}
func watch(flags *v1.OocOptions) error { return iaas.WatchResult(flags, flags.ClusterName) }
