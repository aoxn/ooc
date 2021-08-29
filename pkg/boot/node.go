package boot

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/actions"
	"github.com/aoxn/ooc/pkg/actions/file"
	"github.com/aoxn/ooc/pkg/actions/kubeadm"
	"github.com/aoxn/ooc/pkg/actions/post"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/context"
)

func InitMasterAlone(ctx *context.NodeContext) error {
	node := ctx.NodeObject()
	if node == nil {
		return fmt.Errorf("master, empty node error: %s ", node)
	}
	cfg := node.Status.BootCFG

	nmeta := ctx.NodeMetaData()
	os, err := nmeta.OS()
	if err != nil {
		return fmt.Errorf("init master call meta.OS: %s", err.Error())
	}
	arch, err := nmeta.Arch()
	if err != nil {
		return fmt.Errorf("init master call meta.ARCH: %s", err.Error())
	}
	return actions.RunActions(
		[]actions.Action{
			NewConcurrentPkgDL(&cfg.Spec, os, arch),
			kubeadm.NewActionKubelet(),
			kubeadm.NewActionInit(),
			kubeadm.NewActionCCMAuth(),
			kubeadm.NewActionKubeAuth(),
			post.NewActionPost(),
		},
		// ActionContext is not shareable among different steps.
		// ActionContext is shareable among Actions.
		// share context through ActionContext.SharedOperatorContext among steps.
		actions.NewActionContext(ctx),
	)
}

func InitWorker(ctx *context.NodeContext) error {
	node := ctx.NodeObject()
	if node == nil {
		return fmt.Errorf("empty node error: %s ", node)
	}
	cfg := node.Status.BootCFG

	nmeta := ctx.NodeMetaData()
	os, err := nmeta.OS()
	if err != nil {
		return fmt.Errorf("init worker call meta.OS: %s", err.Error())
	}
	arch, err := nmeta.Arch()
	if err != nil {
		return fmt.Errorf("init worker call meta.ARCH: %s", err.Error())
	}

	return actions.RunActions(
		[]actions.Action{
			NewConcurrentPkgDL(&cfg.Spec, os, arch),
			kubeadm.NewActionKubelet(),
			kubeadm.NewActionJoin(),
		},
		// ActionContext is not shareable among different step.
		// ActionContext is shareable among Actions.
		actions.NewActionContext(ctx),
	)
}

func NewConcurrentPkgDL(
	cfg *v1.ClusterSpec,
	os, arch string,
) actions.Action {

	return actions.NewConcurrentAction(
		// Concurrent Transfer download
		[]actions.Action{
			file.NewAction(
				[]file.File{
					{
						VersionedPath: file.Path{
							Namespace: cfg.Namespace,
							Version:   cfg.Kubernetes.Version,
							Pkg:       file.PKG_KUBERNETES,
							CType:     cfg.CloudType,
							Ftype:     file.FILE_BINARY,
							Project:   "ack",
							OS:        os,
							Arch:      arch,
						},
						CacheDir: fmt.Sprintf("pkg/%s/", file.PKG_KUBERNETES),
					},
				},
			),
			file.NewAction(
				[]file.File{
					{
						VersionedPath: file.Path{
							Namespace:   cfg.Namespace,
							Version:     "0.8.6",
							Pkg:         file.PKG_CNI,
							CType:       cfg.CloudType,
							Ftype:       file.FILE_BINARY,
							Project:     "ack",
							OS:          os,
							Arch:        arch,
							Destination: "/opt/cni/bin/",
						},
					},
				},
			),
		},
	)
}
