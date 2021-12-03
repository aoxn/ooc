// +build linux darwin

package kubeadm

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/actions"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/aoxn/ooc/pkg/utils/cmd"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

const (
	ObjectName        = "config"
	KUBELET_UNIT_FILE = "/etc/systemd/system/kubelet.service"
)

type ActionKubelet struct {
}

// NewAction returns a new ActionInit for kubeadm init
func NewActionKubelet() actions.Action {
	return &ActionKubelet{}
}

// Execute runs the ActionInit
func (a *ActionKubelet) Execute(ctx *actions.ActionContext) error {

	node := ctx.NodeObject()
	if node == nil {
		return fmt.Errorf("node info nil: ActionKubelet")
	}
	switch ctx.OocFlags().Role {
	case v1.NODE_ROLE_WORKER:
		klog.Infof("skip config cert for worker node")
	case v1.NODE_ROLE_MASTER, v1.NODE_ROLE_HYBRID:
		klog.Info("try load kubernetes ca cert")
		if err := LoadCert(node); err != nil {
			return fmt.Errorf("sign: %s", err.Error())
		}
	}
	ip, err := utils.GetDNSIP(node.Status.BootCFG.Spec.Network.SVCCIDR, 10)
	if err != nil {
		return fmt.Errorf("get cluster dns ip fail %s", err.Error())
	}
	if err := ioutil.WriteFile(
		KUBELET_UNIT_FILE,
		[]byte(KubeletUnitFile(node, ip.String())),
		0644,
	); err != nil {
		return fmt.Errorf("write file %s: %s", KUBELET_UNIT_FILE, err.Error())
	}
	err = cmd.Systemctl([]string{"enable", "kubelet"})
	if err != nil {
		return fmt.Errorf("systecmctl enable kubelet error,%s ", err.Error())
	}
	err = cmd.Systemctl([]string{"daemon-reload"})
	if err != nil {
		return fmt.Errorf("systecmctl enable kubelet error,%s ", err.Error())
	}
	err = cmd.Systemctl([]string{"restart", "kubelet"})
	if err != nil {
		return fmt.Errorf("systecmctl start kubelet error,%s ", err.Error())
	}

	//TODO:
	return nil
}

func LoadCert(node *v1.Master) error {
	err := os.MkdirAll("/etc/kubernetes/pki", 0755)
	if err != nil {
		return fmt.Errorf("mkdir error: %s", err.Error())
	}
	apica := append(
		node.Status.BootCFG.Spec.Kubernetes.RootCA.Cert,
	)
	if node.Status.BootCFG.Spec.Kubernetes.ControlRoot != nil {
		apica = append(apica, node.Status.BootCFG.Spec.Kubernetes.ControlRoot.Cert...)
	}
	for name, v := range map[string][]byte{
		"front-proxy-ca.crt": node.Status.BootCFG.Spec.Kubernetes.FrontProxyCA.Cert,
		"front-proxy-ca.key": node.Status.BootCFG.Spec.Kubernetes.FrontProxyCA.Key,
		"ca.crt":             node.Status.BootCFG.Spec.Kubernetes.RootCA.Cert,
		"ca.key":             node.Status.BootCFG.Spec.Kubernetes.RootCA.Key,
		"sa.key":             node.Status.BootCFG.Spec.Kubernetes.SvcAccountCA.Key,
		"sa.pub":             node.Status.BootCFG.Spec.Kubernetes.SvcAccountCA.Cert,
		"apiserver-ca.crt":   apica,
		"apiserver-ca.key":   node.Status.BootCFG.Spec.Kubernetes.RootCA.Key,
	} {
		if err := ioutil.WriteFile(certHome(name), v, 0644); err != nil {
			return fmt.Errorf("write file %s: %s", name, err.Error())
		}
	}

	// do cert clean up
	// let kubeadm do the sign work for the rest
	for _, name := range []string{
		"apiserver.crt", "apiserver.key",
		"front-proxy-client.crt", "front-proxy-client.key",
		"apiserver-kubelet-client.crt", "apiserver-kubelet-client.key",
		"../admin.conf", "../controller-manager.conf",
		"../kubelet.conf", "../scheduler.conf",
		"/var/lib/kubelet/pki/",
	} {
		err := os.Remove(certHome(name))
		if err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				continue
			}
			return fmt.Errorf("clean up existing cert fail: %s", err.Error())
		}
	}
	// clean up pki dir for kubelet to renew.
	rm := <-cmd.NewCmd(
		"rm",
		"-rf",
		"/var/lib/kubelet/pki/",
	).Start()
	return cmd.CmdError(rm)
}

func certHome(key string) string {
	return filepath.Join("/etc/kubernetes/pki/", key)
}

func KubeletUnitFile(node *v1.Master, ip string) string {
	up := []string{
		"[Unit]",
		"Description=kubelet: The Kubernetes NodeObject Agent",
		"Documentation=http://kubernetes.io/docs/",
		"",
		"[Service]",
	}
	down := []string{
		"StartLimitInterval=0",
		"Restart=always",
		"RestartSec=15s",
		"[Install]",
		"WantedBy=multi-user.target",
	}
	var (
		mid  []string
		keys []string
	)
	cfg := NewConfigTpl(node, WithNodeName)
	for k, v := range map[string]string{
		"KUBELET_CLUSTER_DNS":      fmt.Sprintf("--cluster-dns=%s", ip),
		"KUBELET_CGROUP_DRIVER":    "--cgroup-driver=systemd",
		"KUBELET_BOOTSTRAP_ARGS":   "--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf",
		"KUBELET_KUBECONFIG_ARGS":  "--kubeconfig=/etc/kubernetes/kubelet.conf",
		"KUBELET_SYSTEM_PODS_ARGS": "--pod-manifest-path=/etc/kubernetes/manifests",
		//"KUBELET_ALLOW_PRIVILEGE":     "--allow-privileged=true",
		"KUBELET_NETWORK_ARGS":        "--network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin",
		"KUBELET_POD_INFRA_CONTAINER": fmt.Sprintf("--pod-infra-container-image=%s/pause-amd64:3.1", node.Status.BootCFG.Spec.Registry),
		"KUBELET_HOSTNAME_OVERRIDE":   fmt.Sprintf("--hostname-override=%s --provider-id=%s", cfg.NodeName, node.Spec.ID),
		"KUBELET_CERTIFICATE_ARGS":    "--anonymous-auth=false --rotate-certificates=true --cert-dir=/var/lib/kubelet/pki",
		"KUBELET_AUTHZ_ARGS":          "--authorization-mode=Webhook --client-ca-file=/etc/kubernetes/pki/ca.crt",
		"KUBELET_SYSTEM_RESERVED":     "--system-reserved=memory=300Mi --kube-reserved=memory=400Mi --eviction-hard=imagefs.available<15%,memory.available<300Mi,nodefs.available<10%,nodefs.inodesFree<5%",
	} {
		keys = append(keys, fmt.Sprintf("$%s", k))
		mid = append(mid, fmt.Sprintf("Environment=\"%s=%s\"", k, v))
	}
	down = append(
		[]string{fmt.Sprintf("ExecStart=/usr/bin/kubelet %s", strings.Join(keys, " "))},
		down...,
	)
	tmp := append(
		append(up, mid...),
		down...,
	)
	return strings.Join(tmp, "\n")
}
