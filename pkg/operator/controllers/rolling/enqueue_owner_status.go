package rolling

import (
	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type EnqueueTaskRequest struct {
	*handler.EnqueueRequestForOwner
}

// Create implements EventHandler
func (e *EnqueueTaskRequest) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	klog.Info("create task event")
	//e.EnqueueRequestForOwner.Create(evt, q)
	return
}

// Update implements EventHandler
func (e *EnqueueTaskRequest) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	oldTask, ok := evt.ObjectOld.(*acv1.Task)
	if !ok {
		klog.Info("skip update task event")
		return
	}
	newTask, ok := evt.ObjectNew.(*acv1.Task)
	if !ok {
		klog.Info("skip update task event")
		return
	}
	// update task won't trigger reconcile unless phase change
	if newTask.Status.Phase == oldTask.Status.Phase {
		//klog.Info("skip update task event, because phase not change",
		//	"new phase", newTask.Status.Phase,
		//		"old phase", oldTask.Status.Phase)
		return
	} /*else {
		klog.Info("update task event, phase change",
			"new phase", newTask.Status.Phase,
			"old phase", oldTask.Status.Phase)
	}*/
	e.EnqueueRequestForOwner.Update(evt, q)
	return
}

// Delete implements EventHandler
func (e *EnqueueTaskRequest) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	klog.Info("delete task event")
	//e.EnqueueRequestForOwner.Delete(evt, q)
	return
}

// Generic implements EventHandler
func (e *EnqueueTaskRequest) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	klog.Info("generic task event")
	e.EnqueueRequestForOwner.Generic(evt, q)
	return
}
