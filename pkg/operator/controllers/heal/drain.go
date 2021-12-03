package heal

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	gerror "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/drain"
)

func Cordon(
	dr *drain.Helper,
	node *v1.Node,
	cordon bool,
) error {
	help := drain.NewCordonHelper(node)
	help.UpdateIfRequired(cordon)
	err, patcherr := help.PatchOrReplace(dr.Client, false)
	if err != nil || patcherr != nil {
		return gerror.NewAggregate([]error{err, patcherr})
	}
	return nil
}

func Drain(dr *drain.Helper, node *v1.Node) error {
	klog.Infof("try cordon node")
	err := Cordon(dr, node, true)
	if err != nil {
		return fmt.Errorf("cordon node: %s", err.Error())
	}
	pods, errs := dr.GetPodsForDeletion(node.Name)
	if errs != nil {
		return gerror.NewAggregate(errs)
	}
	warnings := pods.Warnings()
	if warnings != "" {
		klog.Infof("WARNING: drain %s", warnings)
	}
	err = dr.DeleteOrEvictPods(pods.Pods())
	if err != nil {
		pending, newErrs := dr.GetPodsForDeletion(node.Name)
		if pending != nil {
			pods := pending.Pods()
			if len(pods) != 0 {
				klog.Infof("there are pending "+
					"pods in node %q when an error occurred: %v", node.Name, err)
				for _, pod := range pods {
					klog.Infof("pods: %s/%s", pod.Namespace, pod.Name)
				}
			}
		}
		if newErrs != nil {
			klog.Infof("drain: GetPodsForDeletion, %s", gerror.NewAggregate(newErrs))
		}
		return err
	}
	klog.Info("drain finished")
	return nil
}
