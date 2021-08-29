//go:build linux || darwin
// +build linux darwin

/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package kubeadmin implements the kubeadm join ActionJoin
package kubeadm

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/actions"
	"github.com/aoxn/ooc/pkg/utils/cmd"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

type ActionJoin struct{}

// NewActionJoin returns a new ActionJoin for kubeadm init
func NewActionJoin() actions.Action {
	return &ActionJoin{}
}

// Execute runs the ActionJoin
func (a *ActionJoin) Execute(ctx *actions.ActionContext) error {
	cfg := NewConfigTpl(
		ctx.NodeObject(),
		WithNodeName,
	)
	status := <-cmd.NewCmd(
		"kubeadm", "join",
		// increase verbosity for debugging
		"--v=6",
		// preflight errors are expected, in particular for swap being enabled
		"--ignore-preflight-errors=all",
		"--node-name", cfg.NodeName,
		"--token", cfg.Kubernetes.KubeadmToken,
		"--discovery-token-unsafe-skip-ca-verification",
		fmt.Sprintf("%s:6443", cfg.Endpoint.Intranet),
	).Start()
	if err := cmd.CmdError(status); err != nil {
		return fmt.Errorf("kubeadm join: %s", err.Error())
	}
	return WaitJoin(ctx)
}

func WaitJoin(ctx *actions.ActionContext) error {
	return wait.Poll(
		2*time.Second,
		5*time.Minute,
		func() (done bool, err error) {
			status := <-cmd.NewCmd(
				"kubectl",
				"--kubeconfig", "/etc/kubernetes/kubelet.conf",
				"get", "no",
			).Start()
			if err := cmd.CmdError(status); err != nil {
				klog.Infof("wait for kubeadm join: %s", err.Error())
				return false, nil
			}
			return true, nil
		},
	)
}
