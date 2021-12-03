package recover

import (
	"fmt"
	v12 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/boot"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"os"
)

const mhelp = `
recover cluster from a remote backup
ooc --region cn-hangzhou \
	--access-key-id xxxxx \
	--access-key-secret yyyyy \
	--name kubernetes-ooc-64 \
	--recover-mode node
`

func NewCommand() *cobra.Command {
	flags := &v12.OocOptions{}
	cmd := &cobra.Command{
		Use:   "recover",
		Short: "recover kubernetes master",
		Long:  mhelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := os.MkdirAll("/etc/ooc", 0755)
			if err != nil {
				return fmt.Errorf("make ooc dir: %s", err.Error())
			}
			//return test(flags,cmd,args)
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Region, "region", "cn-hangzhou", "region")
	cmd.Flags().StringVar(&flags.AccessKeyID, "access-key-id", "", "")
	cmd.Flags().StringVar(&flags.AccessSecret, "access-key-secret", "", "")
	cmd.Flags().StringVar(&flags.ClusterName, "name", "", "the cluster to recover")
	cmd.Flags().StringVar(&flags.RecoverMode, "recover-mode", "iaas", "the recover mode, [iaas|node], default iaas")
	return cmd
}

func runE(flags *v12.OocOptions, cmd *cobra.Command, args []string) error {
	flags.BootType = utils.BootTypeRecover
	if flags.AccessKeyID == "" ||
		flags.AccessSecret == "" ||
		flags.ClusterName == "" {
		return fmt.Errorf("access-key-id and access-key-secret and cluster name must be provided")
	}
	klog.Infof("recover mode[%s]", flags.RecoverMode)
	switch flags.RecoverMode {
	case "iaas":

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
