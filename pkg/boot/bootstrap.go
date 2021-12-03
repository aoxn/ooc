package boot

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/actions/post"
	"github.com/aoxn/ooc/pkg/actions/post/addons"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/apiserver"
	"github.com/aoxn/ooc/pkg/apiserver/auth"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

func NewBootStrapServer(
	cfg apiserver.Configuration,
	boot *v1.ClusterSpec,
) *apiserver.Server {
	if cfg.BindAddr == "" {
		cfg.BindAddr = ":32443"
	}

	return &apiserver.Server{
		Config:    cfg,
		CachedCtx: context.NewCachedContext(boot),
		Auth:      &auth.TokenAuthenticator{},
	}
}

func WaitBootrap(ctx *context.CachedContext, cnt int) error {

	err := wait.Poll(
		3*time.Second,
		10*time.Minute,
		func() (done bool, err error) {

			out, err := utils.Kubectl("--kubeconfig", utils.AUTH_FILE,
				"-l", "node-role.kubernetes.io/master=", "get", "no",
			)
			if err != nil {
				klog.Infof("wait for bootstrap master: %s", err.Error())
				return false, nil
			}
			if len(out)-1 != cnt {
				klog.Infof("wait for bootstrap master count: Expected=%d, Current=%d", cnt, len(out)-1)
				return false, nil
			}
			return true, nil
		},
	)
	if err != nil {
		return fmt.Errorf("wait bootstrap: %s", err.Error())
	}
	if err := post.RunOoc(ctx.BootCFG); err != nil {
		return fmt.Errorf("run ooc: %s", err.Error())
	}
	return addons.InstallAddons(ctx.BootCFG, []addons.ConfigTpl{addons.KUBEPROXY_MASTER})
}
