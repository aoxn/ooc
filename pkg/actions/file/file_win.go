//go:build windows
// +build windows

package file

import (
	"github.com/aoxn/ovm/pkg/actions"
	"path/filepath"
)

type action struct {
	files []Transfer
}

// NewAction returns a new action for kubeadm init
func NewAction(files []Transfer) actions.Action {
	return &action{files: files}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {

	return nil
}

func WgetPath(f Transfer) string { return filepath.Join(f.Cache, filepath.Base(f.URI())) }

func UntarPath(f Transfer) string { return filepath.Join(f.Cache, "untar") }
