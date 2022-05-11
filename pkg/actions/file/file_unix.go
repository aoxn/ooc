//go:build linux || darwin
// +build linux darwin

package file

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/actions"
	"k8s.io/klog/v2"
	"sync"

	"github.com/aoxn/wdrip/pkg/utils"
	"os"
)

type action struct {
	files []File
}

// NewAction returns a new action for kubeadm init
func NewAction(files []File) actions.Action {
	return &action{files: files}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	m := ctx.NodeMetaData()
	if m == nil {
		return fmt.Errorf("node meta should not be nil")
	}
	region, err := m.Region()
	if err != nil || region == "" {
		return fmt.Errorf("empty region: %v", err)
	}
	errs := utils.Errors{}
	wait := sync.WaitGroup{}
	for _, f := range a.files {
		klog.Infof("process file object: %+v", f)
		wait.Add(1)
		err := os.MkdirAll(f.cacheDir(), 0755)
		if err != nil {
			return err
		}
		f.Region = region
		install := func(f *File) {
			defer wait.Done()
			err := f.Download()
			if err != nil {
				errs = append(errs, err)
				return
			}

			err = f.Untar()
			if err != nil {
				errs = append(errs, err)
				return
			}
			err = f.Install()
			if err != nil {
				errs = append(errs, err)
				return
			}
		}
		go install(&f)
	}
	klog.Info("wait for Transfer downloading")
	wait.Wait()

	return errs.HasError()
}
