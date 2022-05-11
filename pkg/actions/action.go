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

package actions

import (
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/context"
	"github.com/aoxn/wdrip/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"reflect"
	"strings"
	"sync"
	"time"
)

func NewActionContext(
	ctx *context.NodeContext,
) *ActionContext {

	return &ActionContext{NodeContext: ctx}
}

type Action interface {
	Execute(ctx *ActionContext) error
}

const RETRY_WORD = "please retry"

func DefaultRetry(action Action) Action {
	return WithRetry(
		action,
		wait.Backoff{
			Duration: 1 * time.Second,
			Factor:   2,
			Steps:    4,
		},
		// default retry decider
		[]NeedRetry{
			func(err error) bool {
				if err == nil {
					return false
				}
				return strings.Contains(err.Error(), RETRY_WORD)
			},
		},
	)
}

// WithRetry Excute with retry with given backoff
// policy and retryDeside
func WithRetry(
	action Action,
	backOff wait.Backoff,
	retryOn []NeedRetry,
) Action {
	return &Retryable{
		Action:  action,
		BackOff: backOff,
		RetryOn: retryOn,
	}
}

type Retryable struct {
	Action
	BackOff wait.Backoff
	RetryOn []NeedRetry
}

func (r *Retryable) Execute(ctx *ActionContext) error {
	return wait.ExponentialBackoff(
		r.BackOff,
		func() (done bool, err error) {
			err = r.Execute(ctx)
			for _, need := range r.RetryOn {
				if need(err) {
					klog.Errorf("retry on error: %s", err.Error())
					return false, nil
				}
			}
			return true, err
		},
	)
}

type NeedRetry func(error) bool

// NewConcurrentAction execute actions concurrently
func NewConcurrentAction(rand []Action) Action {
	return &ConcurrentAction{concurrent: rand}
}

type ConcurrentAction struct{ concurrent []Action }

func (u *ConcurrentAction) Execute(ctx *ActionContext) error {
	var errs utils.Errors
	grp := sync.WaitGroup{}
	for _, action := range u.concurrent {
		grp.Add(1)
		go func(act Action) {
			klog.Infof("start to execute actions: %s", reflect.ValueOf(act).Type())
			defer grp.Done()
			err := act.Execute(ctx)
			if err != nil {
				errs = append(errs, err)
				klog.Errorf("run action concurrent error: %s", err.Error())
			}
		}(action)
	}
	klog.Infof("wait for concurrent actions to finish")
	grp.Wait()
	klog.Infof("concurrent actions finished")
	if len(errs) <= 0 {
		return nil
	}
	return errs
}

// ActionContext is data supplied to all actions
type ActionContext struct{ *context.NodeContext }

func (c *ActionContext) Status() int {
	return c.Value("Status").(int)
}

func (c *ActionContext) Config() *v1.ClusterSpec { return &c.NodeObject().Status.BootCFG.Spec }

func RunActions(
	actions []Action,
	ctx *ActionContext,
) error {
	for _, action := range actions {
		klog.Infof("run action: %s", reflect.TypeOf(action))
		err := action.Execute(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
