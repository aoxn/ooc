package cluster

import (
	v1 "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/iaas"
	"github.com/spf13/cobra"
)

func NewCommandDebug() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "debug",
		Long:  "debug system. ",
	}
	cmd.AddCommand(NewCommandRunCMD())
	cmd.AddCommand(NewCommandNodePool())
	return cmd
}

func NewCommandRunCMD() *cobra.Command {
	flags := &v1.WdripOptions{}
	cmd := &cobra.Command{
		Use:   "runcmd",
		Short: "runcmd -n clusterid -i instance-id -c \"ls -lhstr\"",
		Long:  "run command to debug. ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runcmd(flags)
		},
	}
	cmd.Flags().StringVarP(&flags.ClusterName, "name", "n", "", "cluster name")
	cmd.Flags().StringVarP(&cmdLine.InstanceID, "instance", "i", "", "instance id to run command")
	cmd.Flags().StringVarP(&cmdLine.Command, "command", "c", "", "command line")
	return cmd
}

func NewCommandNodePool() *cobra.Command {
	flags := &v1.WdripOptions{}
	cmd := &cobra.Command{
		Use:     "nodepool",
		Aliases: []string{"np"},
		Short:   "nodepool -n clusterid -i instance-id -c \"ls -lhstr\"",
		Long:    "nodepool command to debug. ",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nodepool(flags)
		},
	}
	cmd.Flags().StringVarP(&flags.ClusterName, "name", "n", "", "cluster name")
	cmd.Flags().StringVarP(&cmdLine.NodePoolID, "nodepoolid", "i", "", "nodepool id to list")
	return cmd
}

func runcmd(flags *v1.WdripOptions) error { return iaas.RunCommand(flags, &cmdLine) }

func nodepool(flags *v1.WdripOptions) error { return iaas.NodePoolOeration(flags, &cmdLine) }
