//go:build linux || darwin || windows
// +build linux darwin windows

package kubeadm

import (
	"encoding/base64"
	"fmt"
	"github.com/aoxn/ovm/pkg/actions"
	"github.com/aoxn/ovm/pkg/utils"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"
)

var ccm = `
kind: Config
contexts:
- context:
    cluster: kubernetes
    user: system:cloud-controller-manager
  name: system:cloud-controller-manager@kubernetes
current-context: system:cloud-controller-manager@kubernetes
users:
- name: system:cloud-controller-manager
  user:
    tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{ .AuthCA}}
    server: https://127.0.0.1:6443
  name: kubernetes
`

type ActionCCM struct {
}

// NewAction returns a new ActionInit for kubeadm init
func NewActionCCMAuth() actions.Action {
	return &ActionCCM{}
}

// Execute runs the ActionInit
func (a *ActionCCM) Execute(ctx *actions.ActionContext) error {

	node := ctx.NodeObject()
	if node == nil {
		return fmt.Errorf("node info nil: ActionKubelet")
	}
	klog.Info("try write ccm auth config")
	err := os.MkdirAll("/etc/kubernetes/", 0755)
	if err != nil {
		return fmt.Errorf("ensure dir /etc/kubernetes :%s", err.Error())
	}

	cfg, err := utils.RenderConfig(
		"ccmauthconfig",
		ccm,
		struct {
			AuthCA     string
			IntranetLB string
		}{
			AuthCA:     base64.StdEncoding.EncodeToString(node.Status.BootCFG.Spec.Kubernetes.RootCA.Cert),
			IntranetLB: node.Status.BootCFG.Spec.Endpoint.Intranet,
		},
	)
	if err != nil {
		return fmt.Errorf("render config error: %s", err.Error())
	}
	return ioutil.WriteFile(
		"/etc/kubernetes/cloud-controller-manager.conf", []byte(cfg), 0755)
}
