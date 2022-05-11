package boot

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions"
	"github.com/aoxn/wdrip/pkg/actions/file"
	"github.com/aoxn/wdrip/pkg/actions/runtime"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/context"
)

func InitContainerRuntime(ctx *context.NodeContext) error {
	node := ctx.NodeObject()
	if node == nil {
		return fmt.Errorf("empty node error: %s ", node)
	}
	cfg := node.Status.BootCFG

	nmeta := ctx.NodeMetaData()
	os, err := nmeta.OS()
	if err != nil {
		return fmt.Errorf("init runtime call meta.OS: %s", err.Error())
	}
	arch, err := nmeta.Arch()
	if err != nil {
		return fmt.Errorf("init runtime call meta.ARCH: %s", err.Error())
	}

	switch cfg.Spec.Runtime.Name {
	case "containerd":
		return fmt.Errorf("not supported")
	case file.PKG_DOCKER:
		// default docker
	}
	return initDocker(ctx, &cfg.Spec, os, arch, ctx.WdripFlags().Bucket)
}

func initDocker(
	ctx *context.NodeContext,
	cfg *v1.ClusterSpec,
	os, arch, bucket string,
) error {
	downs := file.NewAction(
		[]file.File{
			{
				VersionedPath: file.Path{
					Project:   "wdrip",
					Pkg:       file.PKG_DOCKER,
					CType:     cfg.CloudType,
					Ftype:     file.FILE_BINARY,
					OS:        os,
					Arch:      arch,
					Namespace: cfg.Namespace,
					Version:   cfg.Runtime.Version,
				},
				Bucket: bucket,
			},
		},
	)

	return actions.RunActions(
		[]actions.Action{
			downs,
			runtime.NewAction(),
		},
		// ActionContext is not shareable among different step.
		// ActionContext is shareable among Actions.
		actions.NewActionContext(ctx),
	)
}
