package addons

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/aoxn/wdrip/pkg/iaas/provider/alibaba"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type ConfigTpl struct {
	UUID                      string
	Name                      string
	Replicas                  string
	Namespace                 string
	Action                    string
	Region                    string
	ImageVersion              string
	KubeDnsClusterIp          string
	CIDR                      string
	Tpl                       string
	ComponentRevision         string
	IngressSlbNetworkType     string
	ProxyMode                 string
	IntranetApiServerEndpoint string

	IPStack         string
	ServiceCIDR     string
	SecurityGroupID string
	PodVswitchId    string
}

func InstallAddons(pctx *provider.Context, spec *v1.ClusterSpec, cfg []ConfigTpl) error {
	addon, err := defaultAddons(pctx, spec, cfg)
	if err != nil {
		return fmt.Errorf("generate default addon: %s", err.Error())
	}
	for k, v := range addon {
		klog.Infof("install addon: [%s]", k)
		err = wait.Poll(
			2*time.Second,
			1*time.Minute,
			func() (done bool, err error) {
				err = utils.ApplyYamlCommon(
					v, utils.AUTH_FILE,
					fmt.Sprintf("/tmp/addon.%s.yml", k),
				)
				if err != nil {
					klog.Errorf("apply addon error: %s", err.Error())
					return false, nil
				}
				return true, nil
			},
		)
		if err != nil {
			return fmt.Errorf("apply addon wait error: %s", err.Error())
		}
	}
	return nil
}

func AddonConfigsTpl() []ConfigTpl {
	return []ConfigTpl{CCM, CORDDNS, TERWAY, INGRESS, KUBEPROXY_MASTER, KUBEPROXY_WORKER, METRICS_SERVER, CSI_PLUGIN, CSI_PROVISION}
}

func DefaultAddons(pctx *provider.Context, spec *v1.ClusterSpec) (map[string]string, error) {
	return defaultAddons(pctx, spec, []ConfigTpl{})
}

func defaultAddons(pctx *provider.Context, spec *v1.ClusterSpec, cfgs []ConfigTpl) (map[string]string, error) {
	var (
		sgid string
		vsw  string
		err  error
	)

	daddons := make(map[string]string)
	if len(cfgs) == 0 {
		cfgs = AddonConfigsTpl()

		sgid = alibaba.SecrityGroup(pctx.Stack())

		vsw, err = pctx.Provider().VSwitchs(pctx)
		if err != nil {
			return nil, errors.Wrapf(err, "get vswitch")
		}
	}
	tmp := strings.Split(spec.Registry, ".")
	if len(tmp) != 4 {
		return daddons, fmt.Errorf("config registry format error: %s, "+
			"must be registry-vpc.${region}.aliyuncs.com/acs", spec.Registry)
	}
	ip, err := utils.GetDNSIP(spec.Network.SVCCIDR, 10)
	if err != nil {
		return daddons, fmt.Errorf("SVCCIDR must be an ip range: %s , "+
			"for 192.168.0.1/16", spec.Network.SVCCIDR)
	}
	klog.Infof("debug spec: %s", utils.PrettyJson(spec))
	for _, cfg := range cfgs {
		cfg.IntranetApiServerEndpoint = fmt.Sprintf("https://%s:6443", spec.Endpoint.Intranet)
		cfg.ProxyMode = "iptables"
		cfg.Namespace = "kube-system"
		cfg.IngressSlbNetworkType = "internet"
		cfg.ComponentRevision = "1111"
		cfg.Region = tmp[1]
		cfg.CIDR = spec.Network.PodCIDR
		klog.Infof("debug podcidr:  [%s][%s]", cfg.CIDR, spec.Network.PodCIDR)
		cfg.Action = "Install"
		cfg.KubeDnsClusterIp = ip.String()
		cfg.ServiceCIDR = spec.Network.SVCCIDR
		cfg.PodVswitchId = vsw
		cfg.SecurityGroupID = sgid
		cfg.UUID = uuid.New().String()

		data, err := utils.RenderConfig(fmt.Sprintf("addon-config.%s", cfg.Name), cfg.Tpl, cfg)
		if err != nil {
			return daddons, fmt.Errorf("render config: %s", err.Error())
		}
		daddons[cfg.Name] = data
	}
	return daddons, nil
}
