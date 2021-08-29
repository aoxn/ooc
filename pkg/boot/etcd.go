package boot

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/actions"
	"github.com/aoxn/ooc/pkg/actions/etcd"
	"github.com/aoxn/ooc/pkg/actions/file"
	"github.com/aoxn/ooc/pkg/context"
)

func InitEtcd(ctx *context.NodeContext) error {
	node := ctx.NodeObject()
	if node == nil {
		return fmt.Errorf("empty node error: %s ", node)
	}
	cfg := node.Status.BootCFG

	nmeta := ctx.NodeMetaData()
	os, err := nmeta.OS()
	if err != nil {
		return fmt.Errorf("init etcd call meta.OS: %s", err.Error())
	}
	arch, err := nmeta.Arch()
	if err != nil {
		return fmt.Errorf("init etcd call meta.ARCH: %s", err.Error())
	}

	files := []file.File{
		{
			VersionedPath: file.Path{
				Namespace: cfg.Spec.Namespace,
				Pkg:       file.PKG_ETCD,
				CType:     cfg.Spec.CloudType,
				Ftype:     file.FILE_BINARY,
				Project:   "ack",
				OS:        os,
				Arch:      arch,
				Version:   cfg.Spec.Etcd.Version,
			},
		},
	}
	downs := file.NewAction(files)

	return actions.RunActions(
		[]actions.Action{downs, etcd.NewAction()},
		// ActionContext is not shareable among different step.
		// ActionContext is shareable among Actions.
		actions.NewActionContext(ctx),
	)
}
