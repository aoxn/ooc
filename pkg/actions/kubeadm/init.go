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

// Package kubeadminit implements the kubeadm init ActionInit
package kubeadm

import (
	"bytes"
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions"
	"github.com/aoxn/wdrip/pkg/actions/kubeadm/tpl"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/aoxn/wdrip/pkg/utils/cmd"
	"html/template"
	"k8s.io/klog/v2"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/aoxn/wdrip/pkg/utils/kustomize"
	"io/ioutil"
	"path/filepath"
)

type ActionInit struct{}

type ConfigTpl struct {
	*v1.Master
	*v1.ClusterSpec
	NodeName      string
	EtcdEndpoints []string
}

type Option func(tpl *ConfigTpl)

func NewConfigTpl(
	node *v1.Master,
	opt ...Option,
) *ConfigTpl {
	cfg := &ConfigTpl{
		Master: node, ClusterSpec: &node.Status.BootCFG.Spec,
	}
	for _, o := range opt {
		o(cfg)
	}
	return cfg
}

func WithNodeName(tpl *ConfigTpl) {
	_, insid := BreakDownIDOrDie(tpl.Master.Spec.ID)
	tpl.NodeName = fmt.Sprintf("%s.%s", tpl.Master.Spec.IP, insid)
}

func WithEtcdEndpoints(tpl *ConfigTpl) {
	var etcds []string
	if tpl.Etcd.Endpoints != "" {
		for _, host := range strings.Split(tpl.Etcd.Endpoints, ",") {
			etcds = append(etcds, fmt.Sprintf("https://%s:2379", host))
		}
	} else {
		if tpl.Master.Spec.IP == "" {
			panic("node ip must be provided")
		}
		klog.Infof("single etcd addresses: %s", tpl.Master.Spec.IP)
		etcds = append(etcds, fmt.Sprintf("https://%s:2379", tpl.Master.Spec.IP))
	}
	klog.Infof("etcd endpoints configured[%t]: use self address %s", len(tpl.Etcd.Endpoints) != 0, etcds)

	tpl.EtcdEndpoints = etcds
}

func BreakDownIDOrDie(id string) (string, string) {
	tpl := strings.Split(id, ".")
	if len(tpl) != 2 {
		panic(fmt.Sprintf("id format error expect {region.instanceid} got %s", id))
	}
	return tpl[0], tpl[1]
}

// NewAction returns a new ActionInit for kubeadm init
func NewActionInit() actions.Action {
	return &ActionInit{}
}

const (
	KUBEADM_CONFIG_DIR = "/etc/kubeadm/"
)

// Execute runs the ActionInit
func (a *ActionInit) Execute(ctx *actions.ActionContext) error {

	kubeadmConfig, err := getKubeadmConfig(ctx.NodeObject())
	if err != nil {
		return errors.Wrap(err, "failed to generate kubeadm config content")
	}

	klog.Infof("Using kubeadm config:%v", utils.PrettyYaml(kubeadmConfig))
	err = os.MkdirAll(KUBEADM_CONFIG_DIR, 0755)
	if err != nil {
		return fmt.Errorf("mkdir %s error: %s", KUBEADM_CONFIG_DIR, err.Error())
	}
	// copy the config to the node
	if err := ioutil.WriteFile(
		filepath.Join(KUBEADM_CONFIG_DIR, "kubeadm.conf"),
		[]byte(kubeadmConfig),
		0755,
	); err != nil {
		return errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	status := <-cmd.NewCmd(
		"kubeadm", "init",
		// preflight errors are expected, in particular for swap being enabled
		"--ignore-preflight-errors=all",
		// specify our generated config file
		fmt.Sprintf("--config=%s", filepath.Join(KUBEADM_CONFIG_DIR, "kubeadm.conf")),
		"--skip-token-print",
		// increase verbosity for debugging
		"--v=6",
	).Start()
	return cmd.CmdError(status)
}

func setOriginalPki(boot *v1.ClusterSpec) error {
	err := os.MkdirAll("/etc/kubernetes/", 0755)
	if err != nil {
		return fmt.Errorf("ensure dir /etc/kubernetes for admin.local:%s", err.Error())
	}
	counts := map[string][]byte{}
	root := boot.Kubernetes.RootCA
	if root != nil {
		counts["ca.crt"] = root.Cert
		counts["ca.key"] = root.Key
	}

	front := boot.Kubernetes.FrontProxyCA
	if front != nil {
		counts["front-proxy-ca.crt"] = front.Cert
		counts["front-proxy-ca.key"] = front.Key
	}

	sa := boot.Kubernetes.SvcAccountCA
	if sa != nil {
		counts["sa.key"] = sa.Key
		counts["sa.pub"] = sa.Cert
	}
	for name, v := range counts {
		if err := ioutil.WriteFile(certHome(name), v, 0644); err != nil {
			return fmt.Errorf("write file %s: %s", name, err.Error())
		}
	}
	return nil
}

// getKubeadmConfig generates the kubeadm config contents for the cluster
// by running data through the template.
func getKubeadmConfig(node *v1.Master) (path string, err error) {
	// generate the config contents
	raw, err := Config(node)
	if err != nil {
		return "", err
	}
	// fix all the patches to have name metadata matching the generated config
	patches, jsonPatches := setPatchNames(
		allPatchesFromConfig(node),
	)
	// apply patches
	// TODO(bentheelder): this does not respect per node patches at all
	// either make patches cluster wide, or change this
	patched, err := kustomize.Build([]string{raw}, patches, jsonPatches)
	if err != nil {
		return "", err
	}
	return removeMetadata(patched), nil
}

// trims out the metadata.name we put in the config for kustomize matching,
// kubeadm will complain about this otherwise
func removeMetadata(kustomized string) string {
	return strings.Replace(
		kustomized,
		`metadata:
  name: config
`,
		"",
		-1,
	)
}

func allPatchesFromConfig(node *v1.Master) (patches []string, jsonPatches []kustomize.PatchJSON6902) {
	//return cfg.Patches, cfg.PatchesJSON6902
	return nil, nil
}

// setPatchNames sets the targeted object name on every patch to be the fixed
// name we use when generating config objects (we have one of each type, all of
// which have the same fixed name)
func setPatchNames(patches []string, jsonPatches []kustomize.PatchJSON6902) ([]string, []kustomize.PatchJSON6902) {
	fixedPatches := make([]string, len(patches))
	fixedJSONPatches := make([]kustomize.PatchJSON6902, len(jsonPatches))
	for i, patch := range patches {
		// insert the generated name metadata
		fixedPatches[i] = fmt.Sprintf("metadata:\nname: %s\n%s", ObjectName, patch)
	}
	for i, patch := range jsonPatches {
		// insert the generated name metadata
		patch.Name = ObjectName
		fixedJSONPatches[i] = patch
	}
	return fixedPatches, fixedJSONPatches
}

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(node *v1.Master) (config string, err error) {
	cfg := NewConfigTpl(
		node,
		WithNodeName,
		WithEtcdEndpoints,
	)
	t, err := template.New("kubeadm-config").Parse(tpl.Tplv1)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, cfg)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}
	return buff.String(), nil
}
