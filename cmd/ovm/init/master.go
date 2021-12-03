package init

import (
	"fmt"
	"github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/boot"
	"github.com/aoxn/ovm/pkg/context"
)

func initmaster(ctx *context.NodeContext) error {
	steps := []boot.Step{
		boot.InitFunc(ctx),
		boot.InitContainerRuntime,
	}
	switch ctx.OvmFlags().Role {
	case v1.NODE_ROLE_MASTER:
		steps = append(
			steps,
			[]boot.Step{
				boot.InitEtcd,
				boot.InitMasterAlone,
			}...,
		)
	case v1.NODE_ROLE_WORKER:
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
