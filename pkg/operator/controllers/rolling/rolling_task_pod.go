package rolling

import (
	"fmt"
	"k8s.io/klog/v2"

	"context"
	alibabacloudv1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TaskStatus struct {
	Actives   int32
	Failed    int32
	Successed int32
}

func NewGlobalPodTask(r *ReconcileRolling, cxt context.Context, rolling *alibabacloudv1.Rolling) *GlobalPodTask {
	return &GlobalPodTask{
		cxt:     cxt,
		client:  r.client,
		rolling: rolling,
	}
}

type GlobalPodTask struct {
	cxt            context.Context
	client         client.Client
	rolling        *alibabacloudv1.Rolling
	nodesToPodsMap map[string]*corev1.Pod
}

type PodStatus string

const (
	PodStatusActive   PodStatus = "active"
	PodStatusFailed   PodStatus = "failed"
	PodStatusFinished PodStatus = "finished"
)

func (t *GlobalPodTask) ShouldRunTask(node *corev1.Node) bool {
	if _, ok := t.nodesToPodsMap[node.Name]; !ok {
		return true
	}
	return false
}

func (t *GlobalPodTask) CleanupTasks() error {
	return nil
}

func (t *GlobalPodTask) StatusStatistics(nodes []*corev1.Node) (int32, int32, int32, error) {
	var err error
	nodesToJobPods, err := t.getNodesToJobPods()
	if err != nil {
		return 0, 0, 0, err
	}
	t.nodesToPodsMap = nodesToJobPods

	actives, failed, finished, _ := t.podStatusStatistics(nodes)
	return actives, failed, finished, nil
}

func (t *GlobalPodTask) RunTask(node *corev1.Node) error {
	rolling := t.rolling
	prefix := utils.GetNamePrefix(t.rolling.Name)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          rollingLabels(t.rolling),
			GenerateName:    prefix,
			Namespace:       t.rolling.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(rolling, alibabacloudv1.SchemeGroupVersion.WithKind("Rolling"))},
		},
	}

	podSpec := rolling.Spec.PodSpec.Template.Spec.DeepCopy()
	pod.Spec = *podSpec
	return t.client.Create(t.cxt, pod)
}

func (t *GlobalPodTask) getJobPods() ([]*corev1.Pod, error) {

	podsList := &corev1.PodList{}

	labelSelector, err := labelSelector(t.rolling)
	if err != nil {
		return nil, err
	}

	err = t.client.List(t.cxt, podsList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     t.rolling.Namespace,
	})
	if err != nil {
		return nil, err
	}
	pods := []*corev1.Pod{}
	for i, _ := range podsList.Items {
		pods = append(pods, &podsList.Items[i])
	}
	return pods, nil

}

func (t *GlobalPodTask) getNodesToJobPods() (map[string]*corev1.Pod, error) {
	pods, err := t.getJobPods()
	if err != nil {
		return nil, err
	}

	// Group Pods by Node name.
	nodeToJobPods := make(map[string]*corev1.Pod)
	for _, pod := range pods {
		nodeName := pod.Spec.NodeName
		nodeToJobPods[nodeName] = pod
	}
	return nodeToJobPods, nil
}

// 返回Pod统计结果，忽略Node已经不存在的Pod
func (t *GlobalPodTask) podStatusStatistics(nodes []*corev1.Node) (int32, int32, int32, []string) {
	job := t.rolling
	nodesMap := make(map[string]*corev1.Node)
	for _, node := range nodes {
		nodesMap[node.Name] = node
	}

	var actives, failed, finished int32
	failedNodes := []string{}

	for nodeName, pod := range t.nodesToPodsMap {
		_, ok := nodesMap[nodeName]
		if !ok {
			klog.Infof("node %s for pod %s in rolling %s is no longer exists", nodeName, pod.Name, fmt.Sprintf("%s/%s", job.Namespace, job.Name))
			continue
		}
		status := t.podStatus(job, pod)
		switch status {
		case PodStatusFinished:
			finished = finished + 1
		case PodStatusActive:
			actives = actives + 1
		case PodStatusFailed:
			klog.Infof("rolling %s: pod %s on node %s is failed", fmt.Sprintf("%s/%s", job.Namespace, job.Name), pod.Name, pod.Spec.NodeName)
			failedNodes = append(failedNodes, nodeName)
			failed = failed + 1
		}
	}
	return actives, failed, finished, failedNodes
}

func (t *GlobalPodTask) podStatus(rolling *alibabacloudv1.Rolling, pod *corev1.Pod) PodStatus {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		return PodStatusFinished
	case corev1.PodRunning:
		if t.isPodFailed(rolling, pod) {
			return PodStatusFailed
		}
		return PodStatusActive
	case corev1.PodPending:
		return PodStatusActive
	case corev1.PodFailed:
		return PodStatusFailed
	}
	return ""
}

//传入一个Running状态的Pod，判断Pod是否失败
// Pod失败条件：
// 1. restartPolicy=Never的Pod，以错误状态退出
// 2. restartPolicy=OnFailure的Pod，当前失败状态且超过最大重启次数
func (t *GlobalPodTask) isPodFailed(rolling *alibabacloudv1.Rolling, pod *corev1.Pod) bool {
	if pod.Spec.RestartPolicy != corev1.RestartPolicyOnFailure {
		return false
	}

	restartCount := int32(0)
	for i := range pod.Status.InitContainerStatuses {
		stat := pod.Status.InitContainerStatuses[i]
		restartCount += stat.RestartCount
	}
	for i := range pod.Status.ContainerStatuses {
		stat := pod.Status.ContainerStatuses[i]
		restartCount += stat.RestartCount
	}

	return restartCount >= rolling.Spec.RestartLimit
}
