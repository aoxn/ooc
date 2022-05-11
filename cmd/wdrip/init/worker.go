package init

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/boot"
	"github.com/aoxn/wdrip/pkg/context"
)

func initworker(ctx *context.NodeContext) error {
	steps := []boot.Step{
		boot.InitFunc(ctx),
		boot.InitContainerRuntime,
	}
	switch ctx.WdripFlags().Role {
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
