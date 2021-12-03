package addons

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type ConfigTpl struct {
	Name 					  string
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
}

func InstallAddons(spec *v1.ClusterSpec, cfg []ConfigTpl) error {
	addon,err := defaultAddons(spec,cfg)
	if err != nil {
		return fmt.Errorf("generate default addon: %s", err.Error())
	}
	for k,v := range addon {
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
	return []ConfigTpl{CCM, CORDDNS, FLANNEL, INGRESS, KUBEPROXY_MASTER,KUBEPROXY_WORKER, METRICS_SERVER}
}

func DefaultAddons(spec *v1.ClusterSpec) (map[string]string, error) { return defaultAddons(spec,[]ConfigTpl{}) }

func defaultAddons(spec *v1.ClusterSpec, cfgs []ConfigTpl) (map[string]string, error) {
	daddons := make(map[string]string)
	if len(cfgs) == 0 {
		cfgs = AddonConfigsTpl()
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
	for _, cfg := range cfgs {
		cfg.IntranetApiServerEndpoint = fmt.Sprintf("https://%s:6443", spec.Endpoint.Intranet)
		cfg.ProxyMode = "iptables"
		cfg.Namespace = "kube-system"
		cfg.IngressSlbNetworkType = "internet"
		cfg.ComponentRevision = "1111"
		cfg.Region = tmp[1]
		cfg.CIDR = spec.Network.PodCIDR
		cfg.Action = "Install"
		cfg.KubeDnsClusterIp = ip.String()

		data, err := utils.RenderConfig("addon-config", cfg.Tpl, cfg)
		if err != nil {
			return daddons, fmt.Errorf("render config: %s", err.Error())
		}
		daddons[cfg.Name] = data
	}
	return daddons, nil
}
