package dev

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/utils"
	logb "github.com/aoxn/ooc/pkg/utils/log"
	"github.com/aoxn/ooc/pkg/utils/unstructed"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/oss"

	rosc "github.com/denverdino/aliyungo/ros/standard"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

var (
	providerName = "dev"
	cfgtpl       = `
accessKey: keyid
accessKeySecret: secret
templateFile: ~/cache 
region: cn-hangzhou
`
)

func init() {
	provider.AddProvider(providerName, NewDev())
}

type OutPut struct {
	Description string
	OutputKey   string
	OutputValue interface{}
}

type Config struct {
	TemplateFile string `json:"template,omitempty" protobuf:"bytes,1,opt,name=template"`
	AccessKey    string `json:"accessKey,omitempty" protobuf:"bytes,2,opt,name=accessKey"`
	AccessSecret string `json:"accessSecret,omitempty" protobuf:"bytes,3,opt,name=accessSecret"`
}

func NewDev() *Devel {
	return &Devel{}
}

func NewDevInitialized(
	spec *v1.ClusterSpec,
) (*Devel, error) {

	dev := NewDev()
	err := dev.Initialize(
		provider.NewContext(spec),
	)
	if err != nil {
		return dev, fmt.Errorf("initialize dev provider: %s",err.Error())
	}
	return dev, nil
}

var _ provider.Interface = &Devel{}

type Devel struct {
	Cfg *Config
	Ros *rosc.Client
	ESS *ess.Client
	ECS *ecs.Client
	OSS *oss.Client
}

func (n *Devel) Initialize(ctx *provider.Context) error {
	region := ctx.BootCFG().Bind.Region
	options := ctx.OocOptions()
	if options.Region != "" {
		region = options.Region
	}
	if region == "" {
		return fmt.Errorf("region must be specified by --region or in spec.BindInfra.Region")
	}
	// write region back
	ctx.BootCFG().Bind.Region = region
	n.Cfg = &Config{}
	err := json.Unmarshal(ctx.BootCFG().Bind.Provider.Value, n.Cfg)
	if err != nil {
		return fmt.Errorf("decode Ros config: %s", err.Error())
	}
	if n.Cfg.AccessKey == "" ||
		n.Cfg.AccessSecret == "" {
		return fmt.Errorf("region| AccessKey | AccessKeySecret")
	}
	n.Ros = rosc.NewROSClient(n.Cfg.AccessKey, n.Cfg.AccessSecret, common.Region(region))
	n.ESS,err = ess.NewClientWithAccessKey(region, n.Cfg.AccessKey, n.Cfg.AccessSecret)
	if err != nil {
		return errors.Wrap(err,"create ess client")
	}
	n.ECS,err = ecs.NewClientWithAccessKey(region,n.Cfg.AccessKey, n.Cfg.AccessSecret)
	if err != nil {
		return errors.Wrap(err,"create ecs client")
	}
	// the F** Word for the oss region
	oregion := oss.Region(fmt.Sprintf("oss-%s",region))
	n.OSS = oss.NewOSSClient(oregion,false, n.Cfg.AccessKey, n.Cfg.AccessSecret,false)
	return nil
}

func (n *Devel) Create(ctx *provider.Context) (*provider.Id, error) {
	tpl := Template
	if n.Cfg.TemplateFile != "" {
		data, err := ioutil.ReadFile(n.Cfg.TemplateFile)
		if err != nil {
			return nil, fmt.Errorf("read Ros template file: %s", err.Error())
		}
		tpl = string(data)
	}
	bootcfg := ctx.BootCFG()
	paras := map[string]string{
		"MasterImageId":       bootcfg.Bind.Image,
		"SSHFlags":            "true",
		"ZoneId":              bootcfg.Bind.ZoneId,
		"KubernetesVersion":   bootcfg.Kubernetes.Version,
		"DockerVersion":       bootcfg.Runtime.Version,
		"EtcdVersion":         bootcfg.Etcd.Version,
		"MasterLoginPassword": "Just4Test",
		"MasterInstanceType":  bootcfg.Bind.Instance,
		"ProxyMode":           bootcfg.Network.Mode,
		"PublicSLB":   		   "true",
	}
	rtpl, err := RenderUserData(ctx,tpl, true)
	if err != nil {
		return nil, fmt.Errorf("render userdata: %s", err.Error())
	}
	klog.Infof("start to create stack: %s", ctx.BootCFG().ClusterID)
	request := &rosc.CreateStackRequest{
		RegionId:  common.Region(ctx.BootCFG().Bind.Region),
		StackName:            ctx.BootCFG().ClusterID,
		TemplateBody:        rtpl,
		DisableRollback: 	true,
		TimeoutInMinutes:     60,
		Parameters:      Transform(paras),
	}

	response, err := n.Ros.CreateStack(request)
	id := &provider.Id{
		Options:    ctx.OocOptions(),
		CreatedAt:  time.Now().Format("2006-01-02T15:04:05"),
		UpdatedAt:  time.Now().Format("2006-01-02T15:04:05"),
	}

	if err != nil {
		return id, fmt.Errorf("create Ros stack: %s", err.Error())
	}
	id.Name = request.StackName
	id.ResourceId = response.StackId
	klog.Infof("stack created: %s", id.Name)

	//out,err := n.WaitStack(ctx, id)
	//if err != nil {
	//	return id, fmt.Errorf("wait stack create: %s",err.Error())
	//}
	return id, nil
}

func Transform(para map[string]string) []rosc.Parameter{
	var result []rosc.Parameter
	for k,v := range para {
		result = append(
			result,
			rosc.Parameter{
				ParameterKey: 	k,
				ParameterValue: v,
			},
		)
	}
	return result
}

func RenderUserData(
	ctx *provider.Context, tpl string,withNotify bool,
) (string, error) {
	uns,err := unstructed.ToUnstructured(tpl)
	if err != nil {
		return "", fmt.Errorf("unstruct template: %s",err)
	}
	data := NewUserData(ctx)
	base := "Resources.k8s_master_sconfig.Properties.UserData"
	if withNotify {
		var mdata,join []interface{}
		psb,err := getLoadBalancerPart()
		if err != nil {
			return "",fmt.Errorf("part loadbalaner: %s",err.Error())
		}
		pno, err := getNotifierPart()
		if err != nil {
			return "",fmt.Errorf("part notifier: %s",err.Error())
		}
		mdata = append(mdata, PrefixPart())
		mdata = append(mdata, psb...)
		mdata = append(mdata, data)
		mdata = append(mdata, pno...)
		join  = append(join, "", mdata)
		err = uns.SetValue(fmt.Sprintf("%s.Fn::Join", base), join)
	} else {
		err = uns.SetValue(base, strings.Join([]string{PrefixPart(),data},"\n"))
	}
	if err != nil {
		return "", fmt.Errorf("set userdata: %s", err.Error())
	}
	return uns.ToJson()
}

func getNotifierPart() ([]interface{},error) {
	var waiter json.RawMessage
	err := json.Unmarshal(
		[]byte(fmt.Sprintf(`{"Fn::GetAtt": ["k8s_master_waiter_handle", "CurlCli"]}`)), &waiter,
	)
	var parts []interface{}
	parts = append(parts, waiter, "\n")
	return parts, err
}

func getLoadBalancerPart() ([]interface{},error) {
	var sb json.RawMessage
	err := json.Unmarshal(
		[]byte(fmt.Sprintf(`{"Fn::GetAtt": ["k8s_master_slb", "IpAddress"]}`)), &sb,
	)
	var parts []interface{}
	var internetlb json.RawMessage
	err = json.Unmarshal(
		[]byte(
			fmt.Sprintf(
			`{"Fn::If": ["create_public_slb",{"Fn::GetAtt": ["k8s_master_slb_internet","IpAddress"]},""]}`,
			),
		), &internetlb,
	)
	parts = append(parts,"export INTRANET_LB=", sb,"\n")
	parts = append(parts,"export INTERNET_LB=", internetlb, "\n")
	return parts, err
}

func (n *Devel) WaitStack(ctx *provider.Context, id *provider.Id) (out *rosc.GetStackResponse, err error) {
	err = wait.Poll(
		10*time.Second,
		10*time.Minute,
		func() (done bool, err error) {
			out, err = n.Ros.GetStack(
				&rosc.GetStackRequest{
					StackId: id.ResourceId,
					RegionId: common.Region(ctx.BootCFG().Bind.Region),
				},
			)
			if err != nil {
				klog.Errorf("retrieve ros stack error: %s", err.Error())
				return false, nil
			}
			switch out.Status {
			case "CREATE_IN_PROGRESS", "CREATE_ROLLBACK_IN_PROGRESS":
				klog.Infof("create ros stack in progress, [%s/%s], [%s] ", id.ResourceId, id.Name, out.Status)
			case "CREATE_FAILED", "CREATE_ROLLBACK_FAILED":
				return true, fmt.Errorf("stack create failed: %s, %s", out.Status, out.StatusReason)
			case "CREATE_ROLLBACK_COMPLETE", "CREATE_COMPLETE":
				klog.Infof("stack create success, %s/%s [%s]", id.ResourceId, id.Name, out.Status)
				return true, nil
			}

			return false, nil
		},
	)
	return out, err
}

func SetEndpoint(
	cfg *v1.ClusterSpec,
	out *rosc.GetStackResponse,
) error {

	intra, err := findEndpoint(out.Outputs,"APIServerIntranet")
	if err != nil && !strings.Contains(err.Error(), "NotFound") {
		return fmt.Errorf("find apiserver intranet: %s",err.Error())
	}
	inter, err := findEndpoint(out.Outputs, "APIServerInternet")
	if err != nil && !strings.Contains(err.Error(), "NotFound"){
		return fmt.Errorf("find apiserver internet: %s",err.Error())
	}
	cfg.Endpoint.Internet = inter
	cfg.Endpoint.Intranet = intra
	return nil
}

func (n *Devel) WatchResult(ctx *provider.Context, id *provider.Id) error {
	pgb := logb.NewPgmbar(
		id.CreatedAt,
		[]logb.Resource{},
	)

	poll := func() (done bool, err error) {
		var events []rosc.Event
		page := common.Pagination{ PageSize:  50 }
		for {
			res, err := n.Ros.ListStackEvents(
				&rosc.ListStackEventsRequest{
					StackId: id.ResourceId,
					RegionId: common.Region(ctx.BootCFG().Bind.Region),
					Pagination: page,
				},
			)
			if err != nil {
				pgb.SetMessageWithTime(
					fmt.Sprintf("call ros stack event: %s", err.Error()), "",
				)
				return false, nil
			}
			events = append(events, res.Events...)
			p := res.NextPage()
			if p == nil {
				// all pages listed
				break
			}
			page = *p
		}
		merged := ToResources(events)
		//fmt.Printf("%+v\n", utils.PrettyYaml(merged))
		pgb.AddEvents(merged)
		stack, err := n.Ros.GetStack(
			&rosc.GetStackRequest{
				StackId: id.ResourceId,
				RegionId: common.Region(ctx.BootCFG().Bind.Region),
			},
		)
		if err != nil {
			pgb.SetMessageWithTime(
				fmt.Sprintf("call get ros stack: %s", err.Error()), "",
			)
			return false, nil
		}

		status := strings.ToUpper(stack.Status)

		pgb.SetMessageWithTime(
			status, findStackEvent(merged, stack.StackName),
		)
		if strings.Contains(status, "FAIL") ||
			strings.Contains(status, "COMPLETE") {
			time.Sleep(2 * time.Second)
			n.PrintStack(ctx, id, ctx.BootCFG())
			return true, nil
		}
		return false, nil
	}
	return wait.Poll(5*time.Second, 10*time.Minute, poll)
}

func findStackEvent(
	events []logb.Resource,
	name   string,
) string {
	for _, ev := range events {
		if ev.ResourceId == name {
			if ev.UpdatedTime != ev.StartedTime{
				return ev.UpdatedTime
			}
			break
		}
	}
	return time.Now().Format("2006-01-02T15:04:05")
}

func (n *Devel) PrintStack(
	ctx *provider.Context,
	id  *provider.Id,
	spec *v1.ClusterSpec,
) {
	resp, err := n.Ros.GetStack(
		&rosc.GetStackRequest{StackId: id.ResourceId},
	)
	if err != nil {
		klog.Errorf("Finished: print stack information, %s", err.Error())
		return
	}
	err = SetEndpoint(spec, resp)
	if err != nil {
		klog.Warningf("set endpoint fail: %s", err.Error())
	}
	klog.Infof("===========================================================")
	klog.Infof("StackName: %s", id.Name)
	klog.Infof("  StackId: %s", id.ResourceId)
	klog.Infof("%s", utils.PrettyYaml(resp.Outputs))
}

func (n *Devel) Delete(ctx *provider.Context, id *provider.Id) error {
	klog.Infof("try to delete %s", id)
	if id.ResourceId == "" {
		return fmt.Errorf("resourceid empty, delete operation failed")
	}
	_, err := n.Ros.DeleteStack(
		&rosc.DeleteStackRequest{StackId: id.ResourceId},
	)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			klog.Infof("stack does not exists: %s, delete complete, %s", id.Name, err.Error())
			return nil
		}
		return fmt.Errorf("delete Ros stack: %s", err.Error())
	}
	return nil
}

func (n *Devel) Load(ctx *provider.Context) ([]*v1.ClusterSpec, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (n *Devel) Save(ctx *provider.Context, id *provider.Id, spec *v1.ClusterSpec) error {
	return nil
}

func findMasterInformation(out interface{}, target string) ([]string, error) {
	mout, err := toMap(out)
	if err != nil {
		return []string{}, err
	}
	v, ok := mout[target]
	if !ok {
		return []string{}, fmt.Errorf("NotFound")
	}
	sv, err := json.Marshal(v)
	if err != nil {
		return []string{}, err
	}
	var masters []string
	err = json.Unmarshal(sv, &masters)
	if err != nil {
		return []string{}, err
	}
	return masters, nil
}

// id:
//   [APIServerIntranet]
//	 [APIServerInternet]
func findEndpoint(out interface{}, id string) (string, error) {
	mout, err := toMap(out)
	if err != nil {
		return "", err
	}
	v, ok := mout[id]
	if !ok {
		return "", fmt.Errorf("NotFound")
	}
	return fmt.Sprintf("%s", v), nil
}

func toMap(out interface{}) (map[string]interface{}, error) {
	mout := make(map[string]interface{})
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
		mout[v.OutputKey] = v.OutputValue
	}
	return mout, nil
}

