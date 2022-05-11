package alibaba

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

func (n *Devel) InstanceDetail(
	ctx *pd.Context, id []string,
) ([]pd.Instance, error) {
	data, _ := json.Marshal(id)
	req := ecs.CreateDescribeInstancesRequest()
	req.InstanceIds = string(data)
	req.RegionId = n.Cfg.Region
	r, err := n.ECS.DescribeInstances(req)
	if err != nil {
		return nil, errors.Wrapf(err, "InstanceDetail")
	}
	var result []pd.Instance
	for _, v := range r.Instances.Instance {
		var mtag []pd.Value
		for _, t := range v.Tags.Tag {
			mtag = append(mtag, pd.Value{Key: t.TagKey, Val: t.TagValue})
		}
		result = append(result, pd.Instance{
			Id:        v.InstanceId,
			Status:    v.Status,
			Tags:      mtag,
			CreatedAt: normalize(v.CreationTime),
			Ip:        strings.Join(v.VpcAttributes.PrivateIpAddress.IpAddress, ","),
		})
	}

	return result, nil
}

func (n *Devel) RunCommand(ctx *pd.Context, id, cmd string) (pd.Result, error) {

	content := base64.StdEncoding.EncodeToString([]byte(cmd))

	commandType := "RunShellScript"
	waitInvocation := func(timeout time.Duration) error {
		mfunc := func() (bool, error) {
			req := ecs.CreateDescribeInvocationsRequest()
			req.InstanceId = id
			req.RegionId = n.Cfg.Region
			// 阿里云垃圾API。擦
			// 该API的InvokeStatus 过滤有问题，需要手动过滤
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
			cnt := 0
			//兼容
			for _, i := range inv.Invocations.Invocation {
				if i.InvocationStatus == "Running" {
					cnt++
					klog.Infof("command [%s.%s.%s] is still in running", id, i.InvokeId, i.CommandId)
				}
			}
			if cnt == 0 {
				return true, nil
			}
			klog.Infof("invocation still in progress: total %d, %s", cnt, id)
			return false, nil
		}
		return wait.PollImmediate(3*time.Second, timeout, mfunc)
	}
	err := waitInvocation(4 * time.Minute)
	if err != nil {
		return pd.Result{}, errors.Wrapf(err, "wait invocation: ")
	}

	// run command
	req := ecs.CreateRunCommandRequest()
	req.Timed = requests.NewBoolean(false)
	req.CommandContent = content
	req.ContentEncoding = "Base64"
	req.Type = commandType
	req.InstanceId = &[]string{id}
	req.KeepCommand = requests.NewBoolean(false)
	klog.Infof("run command: [%s]", cmd)
	rcmd, err := n.ECS.RunCommand(req)
	if err != nil {
		return pd.Result{}, errors.Wrapf(err, "run command")
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
	if err := waitResult(&ivk, 4*time.Minute); err != nil {
		return pd.Result{}, errors.Wrapf(err, "wait command result")
	}
	r := ivk.InvocationResults.InvocationResult
	if len(r) == 0 {
		return pd.Result{}, nil
	}
	output, err := base64.StdEncoding.DecodeString(r[0].Output)
	if err != nil {
		return pd.Result{}, err
	}
	return pd.Result{Status: ivk.InvocationStatus, OutPut: string(output)}, nil
}

func (n *Devel) DeleteECS(ctx *pd.Context, id string) error {
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

func (n *Devel) StopECS(ctx *pd.Context, id string) error {
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
