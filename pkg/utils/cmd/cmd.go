// +build linux darwin

package cmd

import (
	"fmt"
	gcmd "github.com/go-cmd/cmd"
	"k8s.io/klog/v2"
	"strings"
)

func NewCmd(name string, args ...string) *gcmd.Cmd {
	klog.Infof("Debug RunCmd: %s %s", name, strings.Join(args, " "))
	return gcmd.NewCmd(name, args...)
}

func CmdError(sta gcmd.Status) error {
	if len(sta.Stderr) != 0 {
		klog.Infof("Warning: stand error NotEmpty. ")
		klog.Infof("\tExit=%d,", sta.Exit)
		klog.Infof("\tError=%v,", sta.Error)
		klog.Infof("\tStdError=%v,", sta.Stderr)
		klog.Infof("\tStdout=%s", sta.Stdout)
	}
	if sta.Exit == 0 && sta.Error == nil {
		return nil
	}
	return fmt.Errorf("exit=%d, error: %v. stderr: %v, out: %s", sta.Exit, sta.Error, sta.Stderr, sta.Stdout)
}
func Systemctl(ops []string) error {
	cm := NewCmd(
		"systemctl", ops...,
	)
	result := <-cm.Start()
	return result.Error
}
