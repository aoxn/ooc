package alibaba

import (
	"encoding/json"
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/denverdino/aliyungo/common"
	rosc "github.com/denverdino/aliyungo/ros/standard"
)

const (
	StackID = "StackID"
)

func (n *Devel) GetStackOutPuts(ctx *provider.Context, id *api.ClusterId) (map[string]provider.Value, error) {

	if id.Spec.ResourceId == "" {
		if id.Name != "" {
			resp, err := n.Ros.ListStacks(
				&rosc.ListStacksRequest{
					RegionId:  common.Region(n.Cfg.Region),
					StackName: []string{id.Name},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list rsource get stack: %s", err.Error())
			}

			if len(resp.Stacks) == 0 {
				return nil, fmt.Errorf("no stacks found by name: %s", id.Name)
			}

			if len(resp.Stacks) > 1 {
				// TODO: this is a workaround for ListStacks API problem
				found := false
				for _, stack := range resp.Stacks {
					if stack.StackName == id.Name {
						found = true
						id.Spec.ResourceId = stack.StackId
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("multiple ros stacks by name: %s, count=%d", id.Name, len(resp.Stacks))
				}
			} else {
				id.Spec.ResourceId = resp.Stacks[0].StackId
			}
			// continue find stack output by stackid
		}else {
			return nil, fmt.Errorf("id or name must be provided.")
		}
	}
	resp, err := n.Ros.GetStack(
		&rosc.GetStackRequest{
			RegionId: common.Region(n.Cfg.Region),
			StackId:  id.Spec.ResourceId,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list rsource get stack: %s", err.Error())
	}
	resp.Outputs = append(resp.Outputs, rosc.Output{OutputKey: StackID, OutputValue: id.Spec.ResourceId})
	return toActionMap(resp.Outputs)
}

func (n *Devel) GetInfraStack(
	ctx *provider.Context, id *api.ClusterId,
) (map[string]provider.Value, error) {
	stack := make(map[string]provider.Value)
	if id.Spec.ResourceId == "" {
		return stack, fmt.Errorf("EmptyStackID")
	}
	resource, err := n.Ros.ListStackResources(
		&rosc.ListStackResourcesRequest{
			StackId:  id.Spec.ResourceId,
			RegionId: common.Region(n.Cfg.Region),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("find resource k8s_nodes_scaling_rule fail, %s", err.Error())
	}
	for _, resou := range resource.Resources {
		val := provider.Value{
			Key: resou.LogicalResourceId,
			Val: resou.PhysicalResourceId,
		}
		stack[resou.LogicalResourceId] = val
	}
	return stack, nil
}

func toActionMap(out interface{}) (map[string]provider.Value, error) {
	mout := make(map[string]provider.Value)
	sout, err := json.Marshal(out)
	if err != nil {
		return mout, err
	}
	pout := &[]OutPut{}
	err = json.Unmarshal(sout, pout)
	if err != nil {
		return mout, err
	}
	for _, v := range *pout {
		mout[v.OutputKey] = provider.Value{Key: v.OutputKey, Val: v.OutputValue}
	}
	return mout, nil
}
