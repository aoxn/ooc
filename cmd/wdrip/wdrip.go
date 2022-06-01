/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY WDRIP, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package wdrip implements the root wdrip cobra command, and the cli Main()
package wdrip

import (
	"fmt"
	"github.com/aoxn/wdrip/cmd/wdrip/build"
	"github.com/aoxn/wdrip/cmd/wdrip/cluster"
	initpkg "github.com/aoxn/wdrip/cmd/wdrip/init"
	"github.com/aoxn/wdrip/cmd/wdrip/monitor"
	"github.com/aoxn/wdrip/cmd/wdrip/monkey"
	"github.com/aoxn/wdrip/cmd/wdrip/vm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/aoxn/wdrip/cmd/wdrip/bootstrap"
	"github.com/aoxn/wdrip/cmd/wdrip/operator"
	recv "github.com/aoxn/wdrip/cmd/wdrip/recover"
	"github.com/aoxn/wdrip/cmd/wdrip/token"
	"github.com/aoxn/wdrip/cmd/wdrip/version"
)

const defaultLevel = log.InfoLevel

type Flags struct {
	LogLevel string
}

// NewCommand returns a new cobra.Command implementing the root command for wdrip
func NewCommand() *cobra.Command {
	flags := &Flags{}
	cmd := &cobra.Command{
		Use:   "wdrip",
		Short: "wdrip creates and manages infrastructure agnostic Kubernetes clusters",
		Long:  fmt.Sprintf("%s\n%s", version.Logo, "wdrip creates and manages infrastructure agnostic " +
			"Kubernetes clusters and empower strong infrastructure resilience ability and easy recovery"),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVar(
		&flags.LogLevel,
		"loglevel",
		defaultLevel.String(),
		"logrus log level ",
	)
	// add all top level subcommands
	cmd.AddCommand(bootstrap.NewCommand())
	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(token.NewCommand())
	cmd.AddCommand(token.NewCryptCommand())
	cmd.AddCommand(operator.NewCommand())
	cmd.AddCommand(initpkg.NewCommand())
	cmd.AddCommand(build.NewCommand())
	cmd.AddCommand(cluster.NewCommand())
	cmd.AddCommand(cluster.NewCommandDelete())
	cmd.AddCommand(cluster.NewCommandWatch())
	cmd.AddCommand(cluster.NewCommandGet())
	cmd.AddCommand(cluster.NewCommandEdit())
	cmd.AddCommand(cluster.NewCommandConfig())
	cmd.AddCommand(cluster.NewCommandScale())
	cmd.AddCommand(monitor.NewCommand())
	cmd.AddCommand(recv.NewCommand())
	cmd.AddCommand(monkey.NewCommand())
	cmd.AddCommand(vm.NewCommand())
	cmd.AddCommand(cluster.NewCommandDebug())
	return cmd
}

func runE(flags *Flags, cmd *cobra.Command, args []string) error {
	level := log.InfoLevel
	parsed, err := log.ParseLevel(flags.LogLevel)
	if err != nil {
		klog.Warningf("Invalid log level '%s', defaulting to '%s'", flags.LogLevel, level)
	} else {
		level = parsed
	}
	log.SetLevel(level)
	return nil
}

// Run runs the `wdrip` root command
func Run() error {
	return NewCommand().Execute()
}

// Main wraps Run and sets the log formatter
func Main() {
	ctrl.SetLogger(klogr.New())
	if err := Run(); err != nil {
		klog.Errorf("run error: %s", err.Error())
		os.Exit(1)
	}
}
