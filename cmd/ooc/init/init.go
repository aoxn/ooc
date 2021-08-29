package init

import (
	"fmt"
	v12 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/boot"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &v12.OocOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Kubernetes cluster init",
		Long:  "kubernetes cluster init. configuration management, lifecycle management",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := os.MkdirAll("/etc/ooc", 0755)
			if err != nil {
				return fmt.Errorf("make ooc dir: %s", err.Error())
			}
			//return test(flags,cmd,args)
			return runE(flags, cmd, args)
		},
	}
	typs := fmt.Sprintf(
		"%s, %s, %s, %s",
		utils.BootTypeLocal,
		utils.BootTypeRecover,
		utils.BootTypeOperator,
		utils.BootTypeCoord,
	)
	cmd.Flags().StringVar(&flags.Token, "token", "abcd.1234567890", "authentication token")
	cmd.Flags().StringVar(&flags.Addons, "addons", "", "addons to be installed, [* for all], empty for kubeproxyMaster")
	cmd.Flags().StringVar(&flags.Role, "role", "worker", "Etcd|Master|Both|Worker")
	cmd.Flags().IntVar(&flags.ExpectedMasterCnt, "expected-master-count", 3, "expected master count, default to 3")
	cmd.Flags().StringVar(&flags.Endpoint, "endpoint", "http://127.0.0.1:32443", "api endpoint, eg. http://127.0.0.1:32443")
	cmd.Flags().StringVar(&flags.BootType, "boot-type", "local", typs)
	cmd.Flags().StringVar(&flags.Config, "config", "", "cluster config file, use cordinate bootstrap if not provided")
	return cmd
}

func runE(flags *v12.OocOptions, cmd *cobra.Command, args []string) error {
	ctx, err := context.NewNodeContext(*flags)
	if err != nil {
		return fmt.Errorf("build node context error: %s", err.Error())
	}
	return initnode(ctx)
}

func initnode(ctx *context.NodeContext) error {
	steps := []boot.Step{
		boot.InitFunc(ctx),
		boot.InitContainerRuntime,
	}
	switch ctx.OocFlags().Role {
	case v12.NODE_ROLE_ETCD:
		steps = append(
			steps,
			[]boot.Step{
				boot.InitEtcd,
			}...,
		)
	case v12.NODE_ROLE_MASTER:
		steps = append(
			steps,
			[]boot.Step{
				boot.InitMasterAlone,
			}...,
		)
	case v12.NODE_ROLE_HYBRID:
		steps = append(
			steps,
			[]boot.Step{
				boot.InitEtcd,
				boot.InitMasterAlone,
			}...,
		)
	case v12.NODE_ROLE_WORKER:
		steps = append(
			steps,
			[]boot.Step{
				boot.InitWorker,
			}...,
		)
	}
	for i, step := range steps {
		if err := step(ctx); err != nil {
			return fmt.Errorf("steps %d: %s", i, err.Error())
		}
	}
	return nil
}

func test(flags *v12.OocOptions, cmd *cobra.Command, args []string) error {

	ctx, err := context.NewNodeContext(*flags)
	if err != nil {
		return err
	}
	cnode := ctx.BootNodeClient()
	_, err = cnode.Create(
		&v12.Master{
			TypeMeta: v1.TypeMeta{
				Kind:       "Master",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "192.168.0.1",
			},
			Spec: v12.MasterSpec{
				ID: "192.168.0.1.i-xxxxxxx",
				IP: "192.168.0.1",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("registry node: %s", err.Error())
	}

	cred := ctx.BootCredentialClient()

	credo, err := cred.Get("192.168.0.1.i-xxxxxxx")

	if err != nil {
		return fmt.Errorf("get credential: %s", err.Error())
	}

	fmt.Printf(utils.PrettyYaml(ctx))
	fmt.Printf(utils.PrettyYaml(credo))
	return nil
}
