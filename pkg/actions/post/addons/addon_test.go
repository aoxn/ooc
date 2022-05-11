package addons

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/utils"
	"testing"
)

func TestAddon(t *testing.T) {
	cfg := CCM
	cfg.IntranetApiServerEndpoint = fmt.Sprintf("https://%s:6443", "xxxxxx")
	cfg.ProxyMode = "iptables"
	cfg.Namespace = "kube-system"
	cfg.IngressSlbNetworkType = "internet"
	cfg.ComponentRevision = "1111"
	cfg.Region = "cn-hangzhou"
	//cfg.CIDR = "192.168.0.1"
	cfg.Action = "Install"
	cfg.KubeDnsClusterIp = "192.168.0.10"
	cfg.ServiceCIDR = "172.16.0.1/28"
	cfg.PodVswitchId = "vsw-ixxx"
	cfg.SecurityGroupID = "vgroup-id"

	data, err := utils.RenderConfig(fmt.Sprintf("addon-config.%s", cfg.Name), cfg.Tpl, cfg)
	if err != nil {
		t.Fatalf("render fail: %s", err.Error())
	}

	t.Logf("render data: %s", data)
}
