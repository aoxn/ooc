package alibaba

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	v1 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"strings"
)

func SecrityGroup(stack map[string]provider.Value) string {
	sg, ok := stack["k8s_sg"]
	if !ok {
		klog.Warningf("empty security group for [k8s_sg]")
		return ""
	}
	return sg.Val.(string)
}

func Vswitchs(stack map[string]provider.Value) string {
	vs, ok := stack["k8s_vswitch"]
	if !ok {
		klog.Warningf("empty vswitch ids for [k8s_vswitch]")
		return ""
	}
	return vs.Val.(string)
}

func (n *Devel) CreateNodeGroup(ctx *provider.Context, np *v1.NodePool) (*v1.BindID, error) {
	boot := ctx.BootCFG()
	if boot == nil {
		return nil, fmt.Errorf("create nodegroup: miss cluster spec")
	}
	bind := np.Spec.Infra.Bind
	if bind != nil {
		klog.Infof("scaling group "+
			"might be initialized before. generated=%v", np.Spec.Infra.Bind)
	}
	region := boot.Bind.Region
	gname := fmt.Sprintf("scalinggroup-%s", np.Name)
	dreq := ess.CreateDescribeScalingGroupsRequest()
	dreq.RegionId = region
	dreq.ScalingGroupName = gname
	mgroup, err := n.ESS.DescribeScalingGroups(dreq)
	if err != nil {
		return bind, errors.Wrapf(err, "find scaling group %s", gname)
	}
	var sgrpid string
	if len(mgroup.ScalingGroups.ScalingGroup) <= 0 {
		klog.Infof("create scaling group: %s", gname)
		req := ess.CreateCreateScalingGroupRequest()
		req.RegionId = region
		req.ScalingGroupName = gname
		req.MultiAZPolicy = "COST_OPTIMIZED"
		req.VSwitchIds = &[]string{Vswitchs(ctx.Stack())}
		req.MinSize = requests.NewInteger(0)
		req.MaxSize = requests.NewInteger(1000)
		req.DesiredCapacity = requests.NewInteger(np.Spec.Infra.DesiredCapacity)
		response, err := n.ESS.CreateScalingGroup(req)
		if err != nil {
			return bind, errors.Wrapf(err, "create scaling group, %s", np.Name)
		}
		sgrpid = response.ScalingGroupId
		klog.Infof("created scaling group: %s with id %s", gname, sgrpid)
	} else {
		sgrpid = mgroup.ScalingGroups.ScalingGroup[0].ScalingGroupId
		klog.Infof("found existing scaling group with id: %s=%s", gname, sgrpid)
	}

	scfgName := fmt.Sprintf("scalingconfig-%s", np.Name)
	dsreq := ess.CreateDescribeScalingConfigurationsRequest()
	dsreq.ScalingGroupId = sgrpid
	dsreq.RegionId = region
	dsreq.ScalingConfigurationName = &[]string{scfgName}

	cfg, err := n.ESS.DescribeScalingConfigurations(dsreq)
	if err != nil {
		return bind, errors.Wrapf(err, "find scaling configuration, %s", scfgName)
	}

	var scfgid string
	if len(cfg.ScalingConfigurations.ScalingConfiguration) <= 0 {
		klog.Infof("create scaling configuration: %s for group %s", scfgName, sgrpid)
		sreq := ess.CreateCreateScalingConfigurationRequest()
		sreq.RegionId = region
		sreq.ScalingGroupId = sgrpid
		sreq.SecurityGroupId = SecrityGroup(ctx.Stack())
		//sreq.InstanceType = ""
		// cloud_essd|cloud_ssd|cloud_efficiency|cloud 20-500
		sreq.SystemDiskCategory = "cloud_essd"
		sreq.SystemDiskSize = requests.NewInteger(40)
		sreq.ScalingConfigurationName = scfgName

		sreq.ImageId = utils.DefaultImage(np.Spec.Infra.ImageId)
		data, err := NewWorkerUserData(ctx)
		if err != nil {
			return bind, errors.Wrap(err, "build work userdata")
		}
		sreq.UserData = data
		sreq.Cpu = requests.NewInteger(np.Spec.Infra.CPU)
		sreq.Memory = requests.NewInteger(np.Spec.Infra.Mem)
		sreq.Tags = utils.PrettyJson(map[string]string{"ooc.com": np.Name})
		//sreq.RamRoleName = ""
		sreq.KeyPairName = ""
		res, err := n.ESS.CreateScalingConfiguration(sreq)
		if err != nil {
			return bind, errors.Wrapf(err, "create scaling configuration,%s", sgrpid)
		}
		scfgid = res.ScalingConfigurationId
		klog.Infof("created scaling configuration "+
			"for group %s with id %s by name %s", sgrpid, scfgid, scfgName)
	} else {
		scfgid = cfg.ScalingConfigurations.ScalingConfiguration[0].ScalingConfigurationId
	}

	ereq := ess.CreateEnableScalingGroupRequest()
	ereq.ScalingGroupId = sgrpid
	ereq.ActiveScalingConfigurationId = scfgid
	_, err = n.ESS.EnableScalingGroup(ereq)
	if err != nil &&
		!strings.Contains(err.Error(), "IncorrectScalingGroupStatus") {
		return bind, errors.Wrapf(err, "enable scaling group, %s", sgrpid)
	}
	bind = &v1.BindID{
		ScalingGroupId: sgrpid, ConfigurationId: scfgid,
	}
	return bind, nil
}

func (n *Devel) DeleteNodeGroup(ctx *provider.Context, np *v1.NodePool) error {
	region := ""
	bind := np.Spec.Infra.Bind
	if bind == nil {
		klog.Infof("node group does not have bind infra,skip")
		return nil
	}
	//if bind.ConfigurationId != "" {
	//	dreq := ess.CreateDeleteScalingConfigurationRequest()
	//	dreq.RegionId = region
	//	dreq.ScalingConfigurationId = bind.ConfigurationId
	//	_, err := n.ESS.DeleteScalingConfiguration(dreq)
	//	if err != nil && !strings.Contains(err.Error(), "not found"){
	//		return errors.Wrapf(err, "delete scaling configuration, %s", bind.ConfigurationId)
	//	}
	//}
	if bind.ScalingGroupId != "" {
		klog.Infof("delete scaling group: %s", bind.ScalingGroupId)
		sreq := ess.CreateDeleteScalingGroupRequest()
		sreq.RegionId = region
		sreq.ForceDelete = requests.NewBoolean(true)
		sreq.ScalingGroupId = bind.ScalingGroupId
		_, err := n.ESS.DeleteScalingGroup(sreq)
		if err != nil && strings.Contains(err.Error(), "not found") {
			return errors.Wrapf(err, "delete scaling group, %s", bind.ScalingGroupId)
		}
		klog.Infof("delete scaling group %s finished, %v", bind.ScalingGroupId, err)
	}
	return nil
}

func (n *Devel) ModifyNodeGroup(ctx *provider.Context, np *v1.NodePool) error {
	bind := np.Spec.Infra.Bind
	if bind == nil {
		return fmt.Errorf("modify node group: bind empty infra, %s", np.Name)
	}
	req := ess.CreateModifyScalingGroupRequest()
	req.ScalingGroupId = bind.ScalingGroupId
	req.DesiredCapacity = requests.NewInteger(np.Spec.Infra.DesiredCapacity)
	_, err := n.ESS.ModifyScalingGroup(req)
	return err
}
