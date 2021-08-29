package help

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PatchAll    = "All"
	PatchStatus = "Status"
	PatchSpec   = "Spec"
)

type Setter func(copy runtime.Object) (client.Object, error)

// Patch patch object without prefetch needed. Modify object directly
// diff := func(copy runtime.Object) (client.Object,error) {
//      nins := copy.(*v1.Node)
//		nins.Status.Hash = value
//		nins.Status.Phase = phase
//		nins.Status.Reason = reason
//		nins.Status.LastOperateTime = metav1.Now()
//		return nins,nil
//	}
//	return tools.PatchM(mclient,yourObject, diff, tools.PatchStatus)
func Patch(
	mclient client.Client,
	target client.Object,
	setter Setter,
	resource string,
) error {
	err := mclient.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      target.GetName(),
			Namespace: target.GetNamespace(),
		}, target,
	)
	if err != nil {
		return fmt.Errorf("get origin object: %s", err.Error())
	}

	ntarget, err := setter(target.DeepCopyObject())
	if err != nil {
		return fmt.Errorf("get object diff patch: %s", err.Error())
	}
	oldData, err := json.Marshal(target)
	if err != nil {
		return fmt.Errorf("ensure marshal: %s", err.Error())
	}
	newData, err := json.Marshal(ntarget)
	if err != nil {
		return fmt.Errorf("ensure marshal: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, target)
	if patchErr != nil {
		return fmt.Errorf("create merge patch: %s", patchErr.Error())
	}

	if string(patchBytes) == "{}" {
		return nil
	}
	if resource == PatchSpec || resource == PatchAll {
		err := mclient.Patch(
			context.TODO(), ntarget,
			client.RawPatch(types.MergePatchType, patchBytes),
		)
		if err != nil {
			return fmt.Errorf("patch spec: %s", err.Error())
		}
	}

	if resource == PatchStatus || resource == PatchAll {
		return mclient.Status().Patch(
			context.TODO(), ntarget,
			client.RawPatch(types.MergePatchType, patchBytes),
		)
	}
	return nil
}
