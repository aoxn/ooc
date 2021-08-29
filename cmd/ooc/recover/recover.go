package recover

import (
	"fmt"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/boot"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/iaas"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const mhelp = `
recover cluster from a remote backup
ooc --name kubernetes-ooc-64 \
	--recover-mode node
`

func NewCommand() *cobra.Command {
	flags := &api.OocOptions{}
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "recover kubernetes master",
		Long:  mhelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.ClusterName, "name", "", "the cluster to recover")
	cmd.Flags().StringVar(&flags.RecoverMode, "recover-mode", "iaas", "the recover mode, [iaas|node], default iaas")
	return cmd
}

func runE(flags *api.OocOptions, cmd *cobra.Command, args []string) error {
	flags.BootType = utils.BootTypeRecover
	flags.Role = api.NODE_ROLE_MASTER

	klog.Infof("recover mode[%s]", flags.RecoverMode)
	switch flags.RecoverMode {
	case "iaas":
		return iaas.Recover(flags, flags.ClusterName)
	case "node":
		ctx, err := context.NewNodeContext(*flags)
		if err != nil {
			return fmt.Errorf("build node context error: %s", err.Error())
		}
		return doRecover(ctx)
	}
	return nil
}

func doRecover(ctx *context.NodeContext) error {
	steps := []boot.Step{
		boot.InitFunc(ctx),
		boot.InitContainerRuntime,
		boot.InitEtcd,
		boot.InitMasterAlone,
	}
	for i, step := range steps {
		if err := step(ctx); err != nil {
			return fmt.Errorf("steps %d: %s", i, err.Error())
		}
	}
	return nil
}
