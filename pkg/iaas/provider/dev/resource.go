package dev

import (
	"encoding/json"
	"fmt"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/denverdino/aliyungo/common"
	rosc "github.com/denverdino/aliyungo/ros/standard"
)

const (
	StackID = "StackID"
)

func (n *Devel) GetStackOutPuts(ctx *provider.Context, id *provider.Id) (map[string]provider.Value, error) {

	if id.Name != "" {
		resp, err := n.Ros.ListStacks(
			&rosc.ListStacksRequest{
				RegionId:common.Region(ctx.BootCFG().Bind.Region),
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
					id.ResourceId = stack.StackId
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("multiple ros stacks by name: %s, count=%d", id.Name, len(resp.Stacks))
			}
		} else {
			id.ResourceId = resp.Stacks[0].StackId
		}
		// continue find stack output by stackid
	}
	if id.ResourceId != "" {
		resp, err := n.Ros.GetStack(
			&rosc.GetStackRequest{
				RegionId:common.Region(ctx.BootCFG().Bind.Region),
				StackId: id.ResourceId,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("list rsource get stack: %s", err.Error())
		}
		resp.Outputs = append(resp.Outputs, rosc.Output{OutputKey: StackID, OutputValue: id.ResourceId})
		return toActionMap(resp.Outputs)
	}
	return nil, fmt.Errorf("empty stackid: %v", id)
}


func (n *Devel) GetInfraStack(
	ctx *provider.Context, id  *provider.Id,
) (map[string]provider.Value, error) {
	stack := make(map[string]provider.Value)
	if id.ResourceId == "" {
		return stack, fmt.Errorf("EmptyStackID")
	}
	resource, err := n.Ros.ListStackResources(
		&rosc.ListStackResourcesRequest{
			StackId: id.ResourceId,
			RegionId: common.Region(ctx.BootCFG().Bind.Region),
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

