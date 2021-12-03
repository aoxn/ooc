package alibaba

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

func (n *Devel) InstanceDetail(
	ctx *provider.Context, id []string,
) ([]provider.Instance, error) {
	data, _ := json.Marshal(id)
	req := ecs.CreateDescribeInstancesRequest()
	req.InstanceIds = string(data)
	req.RegionId = n.Cfg.Region
	r, err := n.ECS.DescribeInstances(req)
	if err != nil {
		return nil, errors.Wrapf(err, "InstanceDetail")
	}
	var result []provider.Instance
	for _, v := range r.Instances.Instance {
		var mtag []provider.Value
		for _, t := range v.Tags.Tag {
			mtag = append(mtag, provider.Value{Key: t.TagKey, Val: t.TagValue})
		}
		result = append(result, provider.Instance{
			Id:        v.InstanceId,
			Status:    v.Status,
			Tags:      mtag,
			CreatedAt: normalize(v.CreationTime),
			Ip:        strings.Join(v.VpcAttributes.PrivateIpAddress.IpAddress, ","),
		})
	}

	return result, nil
}

func (n *Devel) RunCommand(ctx *provider.Context, id, cmd string) error {

	content := base64.StdEncoding.EncodeToString([]byte(cmd))

	commandType := "RunShellScript"
	waitInvocation := func(timeout time.Duration) error {
		mfunc := func() (bool, error) {
			req := ecs.CreateDescribeInvocationsRequest()
			req.InstanceId = id
			req.RegionId = n.Cfg.Region
			req.InvokeStatus = "Running"
			req.CommandType = commandType
			inv, err := n.ECS.DescribeInvocations(req)
			if err != nil {
				klog.Errorf("descirbe invocation: %s", err.Error())
				return false, nil
			}
			if inv.TotalCount == 0 {
				return true, nil
			}
			klog.Infof("invocation still in progress: total %s", inv.TotalCount)
			return false, nil
		}
		return wait.PollImmediate(3*time.Second, timeout, mfunc)
	}
	err := waitInvocation(4 * time.Minute)
	if err != nil {
		return errors.Wrapf(err, "wait invocation: ")
	}

	// run command
	req := ecs.CreateRunCommandRequest()
	req.Timed = requests.NewBoolean(false)
	req.CommandContent = content
	req.ContentEncoding = "Base64"
	req.Type = commandType
	req.InstanceId = &[]string{id}
	req.KeepCommand = requests.NewBoolean(false)

	rcmd, err := n.ECS.RunCommand(req)
	if err != nil {
		return errors.Wrapf(err, "run command")
	}

	waitResult := func(ivk *ecs.Invocation, timeout time.Duration) error {
		mfunc := func() (bool, error) {
			req := ecs.CreateDescribeInvocationResultsRequest()
			req.InstanceId = id
			req.InvokeId = rcmd.InvokeId
			inv, err := n.ECS.DescribeInvocationResults(req)
			if err != nil {
				klog.Errorf("describe run command result: %s", err.Error())
				return false, nil
			}
			result := inv.Invocation.InvocationResults.InvocationResult
			if len(result) == 0 {
				klog.Infof("invokeid not found: %s, %s", id, rcmd.InvokeId)
				return false, nil
			}
			if result[0].InvokeRecordStatus != "Running" {
				*ivk = inv.Invocation
				return true, nil
			}
			log.Infof("wait run command, in progress: %s, %s", id, rcmd.InvokeId)
			return false, nil
		}
		return wait.PollImmediate(4*time.Second, timeout, mfunc)
	}
	ivk := ecs.Invocation{}
	return waitResult(&ivk, 4*time.Minute)
}

func (n *Devel) DeleteECS(ctx *provider.Context, id string) error {
	if id == "" {
		return fmt.Errorf("instance id must be provided")
	}
	dreq := ecs.CreateDescribeInstanceAttributeRequest()
	dreq.InstanceId = id
	instance, err := n.ECS.DescribeInstanceAttribute(dreq)
	if err != nil {
		if strings.Contains(err.Error(), "not exist") {
			log.Infof("ecs not found, %s, delete finished", id)
			return nil
		}
		return fmt.Errorf("get ecs status failed: %s", err.Error())
	}
	if instance.Status == "Running" {
		req := ecs.CreateStopInstanceRequest()
		req.InstanceId = id
		req.ForceStop = requests.NewBoolean(true)
		// stop instance in running state
		_, err := n.ECS.StopInstance(req)
		if err != nil {
			return fmt.Errorf("stop instance[%s] error: %s", id, err.Error())
		}
		err = WaitECS(n.ECS, id, "Stopped", StopECSTimeout)
		if err != nil {
			return fmt.Errorf("wait for instance stop: %s, %s", id, err.Error())
		}
	}
	req := ecs.CreateDeleteInstanceRequest()
	req.InstanceId = id
	req.RegionId = n.Cfg.Region
	req.Force = requests.NewBoolean(true)
	_, err = n.ECS.DeleteInstance(req)
	return err
}

func (n *Devel) StopECS(ctx *provider.Context, id string) error {
	if id == "" {
		return fmt.Errorf("instance id must be provided")
	}
	dreq := ecs.CreateDescribeInstanceAttributeRequest()
	dreq.InstanceId = id
	instance, err := n.ECS.DescribeInstanceAttribute(dreq)
	if err != nil {
		if strings.Contains(err.Error(), "not exist") {
			log.Infof("ecs not found, %s, delete finished", id)
			return nil
		}
		return fmt.Errorf("get ecs status failed: %s", err.Error())
	}
	if instance.Status == "Running" {
		klog.Infof("trying to stop ecs: %s", id)
		req := ecs.CreateStopInstanceRequest()
		req.InstanceId = id
		req.ForceStop = requests.NewBoolean(true)
		// stop instance in running state
		_, err := n.ECS.StopInstance(req)
		if err != nil {
			return fmt.Errorf("stop instance[%s] error: %s", id, err.Error())
		}
		return WaitECS(n.ECS, id, "Stopped", StopECSTimeout)
	} else {
		klog.Infof("ecs in [%s] status, skip stop", instance.Status)
	}
	return err
}
