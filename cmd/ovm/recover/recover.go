package recover

import (
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/boot"
	"github.com/aoxn/ovm/pkg/context"
	"github.com/aoxn/ovm/pkg/iaas"
	"github.com/aoxn/ovm/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

const mhelp = `
recover cluster from a remote backup
ovm --name kubernetes-ovm-64 \
	--recover-mode node
`

func NewCommand() *cobra.Command {
	flags := &api.OvmOptions{}
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "recover kubernetes master",
		Long:  mhelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVarP(&flags.ClusterName, "name", "n","", "the cluster to recover")
	cmd.Flags().StringVar(&flags.Bucket, "bucket", "host-ovm", "download package from bucket")
	cmd.Flags().StringVarP(&flags.RecoverFrom, "recover-from-cluster", "f","", "recover from backups")
	cmd.Flags().StringVar(&flags.RecoverMode, "recover-mode", "iaas", "the recover mode, [iaas|node], default iaas")
	return cmd
}

func runE(flags *api.OvmOptions, cmd *cobra.Command, args []string) error {
	flags.BootType = utils.BootTypeRecover
	flags.Role = api.NODE_ROLE_MASTER

	if flags.ClusterName == "" {
		return fmt.Errorf("cluster name must not be empty [-n]")
	}
	if flags.RecoverFrom == "" {
		flags.RecoverFrom = flags.ClusterName
	}
	klog.Infof("recover mode[%s]", flags.RecoverMode)
	switch flags.RecoverMode {
	case "iaas":
		return iaas.Recover(flags)
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
