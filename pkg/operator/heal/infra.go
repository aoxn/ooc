package heal

import (
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	pd "github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/iaas/provider/alibaba"
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Infra interface {
	ControlPlaneECS() (map[string]pd.Instance, error)
	NodePoolECS(nps api.NodePool) (map[string]pd.Instance, error)
}

type InfraManager struct {
	spec  *api.Cluster
	stack map[string]pd.Value
	Infra pd.Interface
}

func NewInfraManager(
	cluster *api.Cluster,
	prvd pd.Interface,
) (Infra,error){
	var err error
	infra := &InfraManager{spec: cluster, Infra: prvd}
	if infra.spec.Spec.Bind.ResourceId == "" {
		resource, err := prvd.GetStackOutPuts(
			pd.NewContextWithCluster(&infra.spec.Spec),
			&api.ClusterId{ObjectMeta: metav1.ObjectMeta{Name: infra.spec.Spec.ClusterID}},
		)
		if err != nil {
			return infra,errors.Wrap(err, "provider: list resource")
		}
		infra.spec.Spec.Bind.ResourceId = resource[alibaba.StackID].Val.(string)
	}
	cctx := pd.NewContextWithCluster(&infra.spec.Spec)
	infra.stack, err = h.LoadStack(prvd, cctx, infra.spec)
	return infra, err
}

func (i *InfraManager) ControlPlaneECS() (map[string]pd.Instance, error) {
	cctx := pd.NewContextWithCluster(&i.spec.Spec).WithStack(i.stack)
	detail, err := i.Infra.ScalingGroupDetail(cctx, "", pd.Option{Action: "InstanceIDS"})
	if err != nil {
		return nil, fmt.Errorf("scaling group: %s", err.Error())
	}
	return detail.Instances, nil
}

func (i *InfraManager) NodePoolECS(nps api.NodePool) (map[string]pd.Instance, error) {
	bind := nps.Spec.Infra.Bind
	if bind == nil {
		return nil, nil
	}
	cctx := pd.NewContextWithCluster(&i.spec.Spec).WithStack(i.stack)

	detail, err := i.Infra.ScalingGroupDetail(
		cctx, bind.ScalingGroupId,
		pd.Option{Action: alibaba.ActionInstanceIDS},
	)
	if err != nil {

		return detail.Instances,errors.Wrapf(err, "group %s detail", bind.ScalingGroupId)
	}
	return detail.Instances, nil
}

