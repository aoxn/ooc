/*
Copyright 2018 The Kubernetes Authors.

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

// Package bootstrap implements the `bootstrap` command
package bootstrap

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/apiserver"
	"github.com/aoxn/ooc/pkg/boot"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"io/ioutil"
	"k8s.io/klog/v2"
)

type flagpole struct {
	BindAddr     string
	Token        string
	MetaConfig   string
	InitialCount int
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a Kubernetes cluster",
		Long:  "It plays a coordinator role when bootstrap a kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Token, "token", "abcd.1234567890", "authentication token")
	cmd.Flags().StringVar(&flags.BindAddr, "bind-addr", "", "bind address")
	cmd.Flags().IntVar(&flags.InitialCount, "initial-count", 3, "initial master count for bootstrap")
	cmd.Flags().StringVar(&flags.MetaConfig, "bootcfg", "", "bootstrap cluster config file")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	cfg := apiserver.Configuration{
		BindAddr: flags.BindAddr,
	}

	server := boot.NewBootStrapServer(cfg, loadBootCfgOrDie(flags.MetaConfig))
	if err := server.Start(); err != nil {
		panic(fmt.Sprintf("run apiserver: %s", err.Error()))
	}
	LogCache(server)

	defer klog.Infof("server shutdown automatically after 20 minutes. %s", cfg.BindAddr)
	return boot.WaitBootrap(server.CachedCtx, flags.InitialCount)
}

func LogCache(v *apiserver.Server) {
	v.CachedCtx.Nodes.Range(
		func(key, value interface{}) bool {
			k := key.(string)
			klog.Infof("==============================================================")
			klog.Infof(k)
			klog.Infof("%s\n\n", utils.PrettyYaml(value))
			return true
		},
	)
}

func loadBootCfgOrDie(filen string) *v1.ClusterSpec {
	cont, err := ioutil.ReadFile(filen)
	if err != nil {
		boot := &v1.ClusterSpec{
			Etcd: v1.Etcd{
				Unit: v1.Unit{
					Name:    "etcd",
					Version: "3.3.8",
				},
			},
			Kubernetes: v1.Kubernetes{
				Unit: v1.Unit{
					Name:    "kubernetes",
					Version: "1.12.6-aliyun.1",
				},
			},
			Runtime: v1.ContainerRuntime{
				Unit: v1.Unit{
					Name:    "runtime",
					Version: "18.09.2",
				},
			},
			Network: v1.Network{
				Mode:    "iptables",
				PodCIDR: "192.168.0.1/16",
				SVCCIDR: "172.16.10.10/20",
				Domain:  "cluster.local",
			},
		}
		setDefaultCredential(filen, boot)
		klog.Errorf("using default bootcfg: %s", utils.PrettyYaml(boot))
		return boot
	}
	bootcfg := &v1.ClusterSpec{}
	err = yaml.Unmarshal(cont, bootcfg)
	if err != nil {
		panic(fmt.Sprintf("bootcfg file provided,but not valid %s", err.Error()))
	}
	fmt.Printf("BOOTCNFG, ONBOOT:\n%s\n", utils.PrettyYaml(bootcfg))
	setDefaultCredential(filen, bootcfg)
	return bootcfg
}

func setDefaultCredential(file string, spec *v1.ClusterSpec) {
	utils.SetDefaultCredential(spec)
	err := ioutil.WriteFile(file, []byte(utils.PrettyYaml(spec)), 0755)
	if err != nil {
		panic(fmt.Sprintf("write back file error: %s", err.Error()))
	}
}
