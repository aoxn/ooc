package boot

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions"
	"github.com/aoxn/wdrip/pkg/actions/file"
	"github.com/aoxn/wdrip/pkg/actions/kubeadm"
	"github.com/aoxn/wdrip/pkg/actions/post"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/context"
	"github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/pkg/errors"
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
	oflag := ctx.WdripFlags()
	// we use ~/.wdrip/config to initializing provider
	pctx, err := provider.NewContext(&oflag, &cfg.Spec)
	if err != nil {
		return errors.Wrap(err, "initialize provider")
	}
	ctx.SetKV(context.ProviderCtx, pctx)

	return actions.RunActions(
		[]actions.Action{
			NewConcurrentPkgDL(&cfg.Spec, os, arch, oflag.Bucket),
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
	oflag := ctx.WdripFlags()
	return actions.RunActions(
		[]actions.Action{
			NewConcurrentPkgDL(&cfg.Spec, os, arch, oflag.Bucket),
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
	os, arch, bucket string,
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
							Project:   "wdrip",
							OS:        os,
							Arch:      arch,
						},
						CacheDir: fmt.Sprintf("pkg/%s/", file.PKG_KUBERNETES),
						Bucket:   bucket,
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
							Project:     "wdrip",
							OS:          os,
							Arch:        arch,
							Destination: "/opt/cni/bin/",
						},
						Bucket: bucket,
					},
				},
			),
		},
	)
}
