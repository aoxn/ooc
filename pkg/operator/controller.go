package operator

import (
	"github.com/aoxn/wdrip/pkg/context/shared"
	"github.com/aoxn/wdrip/pkg/operator/controllers/addon"
	"github.com/aoxn/wdrip/pkg/operator/controllers/master"
	"github.com/aoxn/wdrip/pkg/operator/controllers/nodepool"
	"github.com/aoxn/wdrip/pkg/operator/controllers/noderepair"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var Funcs []func(manager.Manager, *shared.SharedOperatorContext) error

func init() {

	Funcs = append(Funcs, master.AddMaster)
	Funcs = append(Funcs, master.AddMasterSet)
	Funcs = append(Funcs, addon.Add)
	Funcs = append(Funcs, master.AddNode)

	Funcs = append(Funcs, nodepool.AddNodePoolController)
	Funcs = append(Funcs, noderepair.AddNodeRepair)
}

func AddControllers(
	mgr manager.Manager,
	oper *shared.SharedOperatorContext,
) error {
	for _, f := range Funcs {
		err := f(mgr, oper)
		if err != nil {
			return err
		}
	}
	return nil
}
