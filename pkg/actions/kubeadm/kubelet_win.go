//go:build windows
// +build windows

package kubeadm

import (
	"github.com/aoxn/ooc/pkg/actions"
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

	//TODO:
	return nil
}
