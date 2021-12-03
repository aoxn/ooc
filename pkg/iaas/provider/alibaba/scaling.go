package alibaba

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

var (
	ActionUserData = "UserData"
)

func (n *Devel) ModifyScalingConfig(
	ctx *provider.Context, gid string, opt ...provider.Option,
) error {
	stack := ctx.Stack()
	id := stack["k8s_master_sconfig"].Val.(string)
	for _, o := range opt {
		action := ActionUserData
		if o.Action != "" {
			action = o.Action
		}
		switch action {
		case "UserData":
			req := ess.CreateModifyScalingConfigurationRequest()
			req.ScalingConfigurationId = id
			req.UserData = o.Value.Val.(string)
			_, err := n.ESS.ModifyScalingConfiguration(req)
			return err
		}
	}
	panic("implement me")
}

var (
	ActionInstanceIDS = "InstanceIDS"
)

func (n *Devel) ScalingGroupDetail(
	ctx *provider.Context, gid string, opt provider.Option,
) (provider.ScaleGroupDetail, error) {
	stack := ctx.Stack()
	if stack == nil {
		return provider.ScaleGroupDetail{}, fmt.Errorf("stack context must be exist")
	}
	vpcid := stack["k8s_vpc"].Val.(string)
	action := ActionInstanceIDS
	if opt.Action != "" {
		action = opt.Action
	}
	if gid == "" {
		// warning: it is not the best options setting default value to master group
		klog.Infof("scaling group not provided, default to master group")
		gid = stack["k8s_master_sg"].Val.(string)
	}

	result := provider.ScaleGroupDetail{
		GroupId:   gid,
		Instances: make(map[string]provider.Instance),
	}
	ireq := ess.CreateDescribeScalingGroupsRequest()
	ireq.ScalingGroupId = &[]string{gid}
	ireq.RegionId = n.Cfg.Region
	sg, err := n.ESS.DescribeScalingGroups(ireq)
	if err != nil {
		return result, errors.Wrapf(err, "ess api:")
	}
	grps := sg.ScalingGroups.ScalingGroup
	if len(grps) <= 0 {
		return result, fmt.Errorf("sgroupid [%s] not found", gid)
	}
	if grps[0].VpcId != vpcid {
		klog.Errorf("invalid vpcid [%s] for scaling group [%s], expect[%s]", grps[0].VpcId,gid, vpcid)
		return result, fmt.Errorf("InvalidVPC")
	}

	switch action {
	case ActionInstanceIDS:
		req := ess.CreateDescribeScalingInstancesRequest()
		req.RegionId = n.Cfg.Region
		req.ScalingGroupId = gid
		ins, err := n.ESS.DescribeScalingInstances(req)
		if err != nil {
			return result, err
		}
		var mins []string
		for _, i := range ins.ScalingInstances.ScalingInstance {
			mins = append(mins, i.InstanceId)
		}
		data, _ := json.Marshal(mins)
		ireq := ecs.CreateDescribeInstancesRequest()
		ireq.InstanceIds = string(data)
		ireq.RegionId = n.Cfg.Region
		attri, err := n.ECS.DescribeInstances(ireq)
		if err != nil {
			return result, fmt.Errorf("instance detail call: %s", err.Error())
		}
		for _, i := range attri.Instances.Instance {
			is := provider.Instance{
				Id:        i.InstanceId,
				Ip:        strings.Join(i.VpcAttributes.PrivateIpAddress.IpAddress, ","),
				CreatedAt: normalize(i.CreationTime),
				Status:    string(i.Status),
			}
			var mtag []provider.Value
			for _, v := range i.Tags.Tag {
				mtag = append(mtag, provider.Value{Key: v.TagKey, Val: v.TagValue})
			}
			is.Tags = mtag
			result.Instances[i.InstanceId] = is
		}
		return result, nil
	}
	return result, fmt.Errorf("[ScalingGroupDetail] unknown action: %s", action)
}

func normalize(t string) string {
	return strings.Replace(t, "Z", ":30Z", -1)
}

func WaitActivity(
	client *ess.Client,
	id string,
	region string,
) error {
	return wait.PollImmediate(
		10*time.Second, 4*time.Minute,
		func() (done bool, err error) {
			req := ess.CreateDescribeScalingActivitiesRequest()
			req.ScalingGroupId = id
			req.StatusCode = "InProgress"
			req.RegionId = region
			result, err := client.DescribeScalingActivities(req)
			if err != nil {
				klog.Errorf("[ScaleMasterGroup] wait scaling activity to complete: %s", err.Error())
				return false, nil
			}
			klog.Infof("[ScaleMasterGroup] %d activities is InProgress", len(result.ScalingActivities.ScalingActivity))
			if len(result.ScalingActivities.ScalingActivity) != 0 {
				return false, nil
			}
			klog.Infof("[ScaleMasterGroup] all scaling activity finished.")
			return true, nil
		},
	)
}

func (n *Devel) ScaleMasterGroup(
	ctx *provider.Context, gid string, desired int,
) error {
	// load scaling group id from stack
	stack := ctx.Stack()
	sgid := stack["k8s_master_sg"].Val.(string)
	srid := stack["k8s_master_srule"].Val.(string)
	region := n.Cfg.Region

	req := ess.CreateModifyScalingRuleRequest()
	req.RegionId = region
	req.ScalingRuleId = srid
	req.AdjustmentType = "TotalCapacity"
	req.AdjustmentValue = requests.NewInteger(desired)
	_, err := n.ESS.ModifyScalingRule(req)
	if err != nil {
		return fmt.Errorf("set scaling rule to %d fail: %s", desired, err.Error())
	}
	sreq := ess.CreateDescribeScalingRulesRequest()
	sreq.RegionId = region
	sreq.ScalingRuleId = &[]string{srid}
	rules, err := n.ESS.DescribeScalingRules(sreq)
	if err != nil {
		return fmt.Errorf("find scaling rule detail: %s", err.Error())
	}
	if len(rules.ScalingRules.ScalingRule) != 1 {
		return fmt.Errorf("multiple scaling rule found: %d by id %s", len(rules.ScalingRules.ScalingRule), srid)
	}
	klog.Infof("wait for activities: %s", sgid)
	// check for running scalingActivities first.
	// wait for activity available
	// n.ESS.DescribeScalingActivities()
	err = WaitActivity(n.ESS, sgid, string(region))
	if err != nil {
		return fmt.Errorf("wait for activity enable: %s", sgid)
	}

	ereq := ess.CreateExecuteScalingRuleRequest()
	ereq.ScalingRuleAri = rules.ScalingRules.ScalingRule[0].ScalingRuleAri
	_, err = n.ESS.ExecuteScalingRule(ereq)
	if err != nil {
		if !strings.Contains(err.Error(), "IncorrectCapacity.NoChange") {
			return fmt.Errorf("execute scaling rule failed: %s", err.Error())
		}
		klog.Errorf("master group ess "+
			"desired count not change[%d], continue: %s", desired, err.Error())
	}
	klog.Infof("[ScaleMasterGroup] execute scale rule: target=%d, wait for finish", desired)
	// wait for scale finish. success or fail
	return WaitActivity(n.ESS, sgid, string(region))
}

func (n *Devel) ScaleNodeGroup(
	ctx *provider.Context, gid string, desired int,
) error {
	// load scaling group id from stack
	sgid := gid
	region := n.Cfg.Region

	req := ess.CreateDescribeScalingRulesRequest()
	req.RegionId = region
	req.ScalingGroupId = gid
	rules, err := n.ESS.DescribeScalingRules(req)
	if err != nil {
		return fmt.Errorf("find node scaling rule detail: %s", err.Error())
	}
	var rule *ess.ScalingRule
	for _, r := range rules.ScalingRules.ScalingRule {
		if r.ScalingGroupId == gid &&
			r.ScalingRuleName == "NodeScale" {
			rule = &r
			break
		}
	}
	if rule == nil {
		//create rule
		req := ess.CreateCreateScalingRuleRequest()
		req.RegionId = region
		req.ScalingGroupId = gid
		req.ScalingRuleName = "NodeScale"
		req.AdjustmentType = "TotalCapacity"
		req.AdjustmentValue = requests.NewInteger(desired)
		result, err := n.ESS.CreateScalingRule(req)
		if err != nil {
			return fmt.Errorf("create scaling rule [NodeScale] error: %s", err.Error())
		}
		rule = &ess.ScalingRule{
			ScalingRuleAri: result.ScalingRuleAri,
			ScalingGroupId: gid,
			ScalingRuleId:  result.ScalingRuleId,
		}
	} else {
		req := ess.CreateModifyScalingRuleRequest()
		req.RegionId = region
		req.ScalingRuleId = rule.ScalingRuleId
		req.AdjustmentType = "TotalCapacity"
		req.AdjustmentValue = requests.NewInteger(desired)
		_, err := n.ESS.ModifyScalingRule(req)
		if err != nil {
			return fmt.Errorf("set scaling rule to %d fail: %s", desired, err.Error())
		}
	}

	// check for running scalingActivities first.
	// wait for activity available
	// n.ESS.DescribeScalingActivities()
	err = WaitActivity(n.ESS, sgid, string(region))
	if err != nil {
		return fmt.Errorf("wait for activity enable: %s", sgid)
	}
	ereq := ess.CreateExecuteScalingRuleRequest()
	ereq.ScalingRuleAri = rules.ScalingRules.ScalingRule[0].ScalingRuleAri
	_, err = n.ESS.ExecuteScalingRule(ereq)
	if err != nil {
		return fmt.Errorf("execute node scaling rule failed: %s", err.Error())
	}
	klog.Infof("[ScaleNodeGroup] execute scale rule: target=%d, wait for finish", desired)
	// wait for scale finish. success or fail
	return WaitActivity(n.ESS, sgid, string(region))
}

func (n *Devel) RemoveScalingGroupECS(
	ctx *provider.Context, gid string, ecs string,
) error {
	klog.Infof("[RemoveScalingGroupECS] trying to remove scaling group ecs: %s", ecs)
	// load scaling group id from stack
	stack := ctx.Stack()
	sgid := stack["k8s_master_sg"].Val.(string)
	//srid := stack["k8s_master_srule"].Val.(string)
	waitActivity := func(id string) error {
		return wait.PollImmediate(
			10*time.Second, 4*time.Minute,
			func() (done bool, err error) {
				req := ess.CreateDescribeScalingActivitiesRequest()
				req.ScalingGroupId = id
				req.StatusCode = "InProgress"
				req.RegionId = n.Cfg.Region
				result, err := n.ESS.DescribeScalingActivities(req)
				if err != nil {
					klog.Errorf("[RemoveScalingGroupECS] wait scaling activity to complete: %s", err.Error())
					return false, nil
				}
				klog.Infof("[RemoveScalingGroupECS] %d activities is InProgress", len(result.ScalingActivities.ScalingActivity))
				if len(result.ScalingActivities.ScalingActivity) != 0 {
					return false, nil
				}
				klog.Infof("[RemoveScalingGroupECS] all scaling activity finished.")
				return true, nil
			},
		)
	}
	// check for running scalingActivities first.
	// wait for activity available
	// n.ESS.DescribeScalingActivities()
	err := waitActivity(sgid)
	if err != nil {
		return fmt.Errorf("wait for activity enable: %s", sgid)
	}
	klog.Infof("[RemoveScalingGroupECS] remove instance %s, wait for finish", ecs)
	req := ess.CreateRemoveInstancesRequest()
	req.InstanceId = &[]string{ecs}
	req.ScalingGroupId = sgid
	_, err = n.ESS.RemoveInstances(req)
	if err != nil {
		return fmt.Errorf("remove sg instance: %s %s", ecs, err.Error())
	}
	// wait for scale finish. success or fail
	return waitActivity(sgid)
}

func (n *Devel) RestartECS(ctx *provider.Context, id string) error {

	req := ecs.CreateDescribeInstanceAttributeRequest()
	req.InstanceId = id
	instance, err := n.ECS.DescribeInstanceAttribute(req)
	if err != nil {
		return fmt.Errorf("get ecs status failed: %s", err.Error())
	}
	if instance.Status == "Running" {
		// stop instance in running state
		sreq := ecs.CreateStopInstanceRequest()
		sreq.InstanceId = id
		_, err = n.ECS.StopInstance(sreq)
		if err != nil {
			return fmt.Errorf("stop instance[%s] error: %s ", id, err.Error())
		}
		err = WaitECS(n.ECS, id, "Stopped", 120)
		if err != nil {
			return fmt.Errorf("wait for instance stop: %s, %s", id, err.Error())
		}
	}
	sreq := ecs.CreateStartInstanceRequest()
	sreq.InstanceId = id
	_, err = n.ECS.StartInstance(sreq)
	if err != nil {
		return fmt.Errorf("start instance[%s]: %s", id, err.Error())
	}
	return WaitECS(n.ECS, id, "Running", 120)
}

type ReplaceConfig struct {
	Id       string
	UserData string
	ImageId  string
}

func (n *Devel) ReplaceSystemDisk(
	ctx *provider.Context, eid string, userdata string, opt provider.Option,
) error {

	//cfg := opt.Value.Val.(*ReplaceConfig)

	klog.Infof("[ReplaceSystemDisk] try stop intance: %s", eid)
	ids, _ := json.Marshal([]string{eid})
	ereq := ecs.CreateDescribeInstancesRequest()
	ereq.InstanceIds = string(ids)
	iins, err := n.ECS.DescribeInstances(ereq)
	if err != nil {
		return fmt.Errorf("describe instance: %s", eid)
	}
	if len(iins.Instances.Instance) <= 0 {
		return errors.Wrapf(err, "no instance find by id", eid)
	}
	inst := iins.Instances.Instance[0]
	if inst.Status == "Running" {
		klog.Infof("[ReplaceSystemDisk] instance is in [Running] state, try stop %s", eid)
		req := ecs.CreateStopInstanceRequest()
		req.InstanceId = eid
		_, err = n.ECS.StopInstance(req)
		if err != nil {
			return fmt.Errorf("stop instance error: %s", err.Error())
		}
	}
	klog.Infof("[ReplaceSystemDisk] wait instance to stop: %s", eid)
	err = WaitECS(n.ECS, eid, "Stopped", 120)
	if err != nil {
		return fmt.Errorf("wait %s stop: %s", eid, err.Error())
	}

	if len(userdata) != 0 {
		klog.Infof("[ReplaceSystemDisk] update userdata for instance: %s", eid)
		req := ecs.CreateModifyInstanceAttributeRequest()
		req.InstanceId = eid
		req.UserData = userdata
		_, err = n.ECS.ModifyInstanceAttribute(req)
		if err != nil {
			return fmt.Errorf("replace instance userdata: %s", err.Error())
		}
	} else {
		klog.Infof("[ReplaceSystemDisk] no userdata provided, skip userdata update")
	}

	klog.Infof("[ReplaceSystemDisk] replace instance system disk: %s", eid)
	req := ecs.CreateReplaceSystemDiskRequest()
	req.InstanceId = eid
	req.ImageId = inst.ImageId
	_, err = n.ECS.ReplaceSystemDisk(req)
	if err != nil {
		return fmt.Errorf("replace system disk: %s", err.Error())
	}

	if inst.KeyPairName != "" {
		bids, err := json.Marshal([]string{inst.InstanceId})
		if err != nil {
			return fmt.Errorf("get instances error: %s", err.Error())
		}
		req := ecs.CreateAttachKeyPairRequest()
		req.RegionId = inst.RegionId
		req.KeyPairName = inst.KeyPairName
		req.InstanceIds = string(bids)
		_, err = n.ECS.AttachKeyPair(req)
		if err != nil {
			return fmt.Errorf("attach instance keypair: %s", err.Error())
		}
	}

	klog.Infof("[ReplaceSystemDisk] wait for "+
		"instance to be in stopped status: %s , replace system disk", eid)

	err = WaitECS(n.ECS, eid, "Stopped", 120)
	if err != nil {
		return fmt.Errorf("wait instance stop timeout: %s", err.Error())
	}
	klog.Infof("[ReplaceSystemDisk] start instance: %s", eid)
	sreq := ecs.CreateStartInstanceRequest()
	sreq.InstanceId = eid
	_, err = n.ECS.StartInstance(sreq)
	if err != nil {
		return fmt.Errorf("start instance: %s", err.Error())
	}
	klog.Infof("[ReplaceSystemDisk] wait instance to be startd: %s", eid)
	return WaitECS(n.ECS, eid, "Running", 120)
}

func (n *Devel) TagECS(
	ctx *provider.Context, id string, val ...provider.Value,
) error {
	if len(val) == 0 {
		return nil
	}
	var rtags []string
	var atags []ecs.TagResourcesTag
	for _, v := range val {
		atag := ecs.TagResourcesTag{
			Key: v.Key, Value: v.Val.(string),
		}
		rtags = append(rtags, v.Key)
		atags = append(atags, atag)
	}
	req := ecs.CreateUntagResourcesRequest()
	req.RegionId = n.Cfg.Region
	req.ResourceId = &[]string{id}
	req.TagKey = &rtags
	req.ResourceType = "instance"
	_, err := n.ECS.UntagResources(req)
	if err != nil && strings.Contains(err.Error(), "NotFound") {
		return fmt.Errorf("remove tag: %s, %s, %s", err.Error(), id, rtags)
	}

	areq := ecs.CreateTagResourcesRequest()
	areq.ResourceType = "instance"
	areq.RegionId = n.Cfg.Region
	areq.ResourceId = &[]string{id}
	areq.Tag = &atags

	_, err = n.ECS.TagResources(areq)
	return err
}

func Now() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

const (
	StopECSTimeout  = 240
	StartECSTimeout = 300

	InstanceDefaultTimeout = 120
	DefaultWaitForInterval = 5
)

func WaitECS(
	client *ecs.Client,
	id, status string,
	timeout int,
) error {
	if timeout <= 0 {
		timeout = InstanceDefaultTimeout
	}
	req := ecs.CreateDescribeInstanceAttributeRequest()
	req.InstanceId = id
	for {
		instance, err := client.DescribeInstanceAttribute(req)
		if err != nil {
			return err
		}
		if instance.Status == status {
			//TODO
			//Sleep one more time for timing issues
			time.Sleep(DefaultWaitForInterval * time.Second)
			break
		}
		timeout = timeout - DefaultWaitForInterval
		if timeout <= 0 {
			return fmt.Errorf("timeout waiting %s %s", id, status)
		}
		time.Sleep(DefaultWaitForInterval * time.Second)

	}
	return nil
}
