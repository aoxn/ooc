package rolling

import (
	"context"
	//log "github.com/sirupsen/logrus"
	acv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RollingTaskManager interface {
	ShouldRunTask(node *corev1.Node) bool
	StatusStatistics(nodes []*corev1.Node) (int32, int32, int32, error)
	RunTask(node *corev1.Node) error
	CleanupTasks() error
}

func NewRollingTaskManager(r *ReconcileRolling, cxt context.Context, rolling *acv1.Rolling) RollingTaskManager {
	return NewCRDTask(r, cxt, rolling)
}

func NewCRDTask(r *ReconcileRolling, cxt context.Context, rolling *acv1.Rolling) *CRDTask {
	return &CRDTask{
		cxt:     cxt,
		client:  r.client,
		rolling: rolling,
	}
}

type CRDTask struct {
	cxt             context.Context
	client          client.Client
	rolling         *acv1.Rolling
	nodesToTasksMap map[string]*acv1.Task
}

func (t *CRDTask) StatusStatistics(nodes []*corev1.Node) (actives, failed, finished int32, err error) {
	t.nodesToTasksMap, err = t.nodesToTasks(t.cxt, t.rolling)
	if err != nil {
		return
	}

	actives, failed, finished = t.tasksStatistics()

	return
}

func (t *CRDTask) ShouldRunTask(node *corev1.Node) bool {
	if _, ok := t.nodesToTasksMap[node.Name]; !ok {
		return true
	}
	return false
}

func (t *CRDTask) CleanupTasks() error {
	for _, task := range t.nodesToTasksMap {
		for i := 0; i <= 3; i++ {
			if err := t.client.Delete(t.cxt, task); err == nil {
				break
			}
		}
	}
	return nil
}

func (t *CRDTask) RunTask(node *corev1.Node) error {
	rolling := t.rolling
	task := &acv1.Task{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: utils.GetNamePrefix(t.rolling.Name),
			Labels:       rollingLabels(rolling),
			Namespace:    rolling.Namespace,
		},
	}
	task.OwnerReferences = append(task.OwnerReferences, *metav1.NewControllerRef(rolling, acv1.SchemeGroupVersion.WithKind("Rolling")))
	task.Spec = *rolling.Spec.TaskSpec.DeepCopy()
	task.Spec.NodeName = node.Name
	task.Spec.TaskType = acv1.TaskTypeUpgrade
	return t.client.Create(t.cxt, task)
}

func (t *CRDTask) tasksStatistics() (actives, failed, completed int32) {
	for _, task := range t.nodesToTasksMap {
		switch task.Status.Phase {
		case acv1.PhaseReconciling:
			actives++
		case acv1.PhaseCompleted:
			completed++
		case acv1.PhaseFailed:
			failed++
		default:
			actives++
		}
	}

	return
}

func (t *CRDTask) nodesToTasks(cxt context.Context, rolling *acv1.Rolling) (map[string]*acv1.Task, error) {
	tasks := &acv1.TaskList{}

	labelSelector, err := labelSelector(rolling)
	if err != nil {
		return nil, err
	}

	err = t.client.List(cxt, tasks, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     rolling.Namespace,
	})
	if err != nil {
		return nil, err
	}
	taskToNode := make(map[string]*acv1.Task)
	for i, _ := range tasks.Items {
		task := tasks.Items[i]
		if task.Spec.NodeName != "" {
			taskToNode[task.Spec.NodeName] = &task
		}
	}
	return taskToNode, nil
}
