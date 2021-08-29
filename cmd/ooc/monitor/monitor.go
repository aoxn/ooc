package monitor

import (
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/spf13/cobra"
)

const mhelp = `
monitor cluster from a remote backup
ooc --name kubernetes-ooc-64 \
	--monitor-mode node
`

func NewCommand() *cobra.Command {
	flags := &api.OocOptions{}
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "monitor kubernetes master",
		Long:  mhelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVarP(&flags.ClusterName, "name", "n", "", "the cluster to monitor")
	return cmd
}

func runE(flags *api.OocOptions, cmd *cobra.Command, args []string) error {

	return nil
}
