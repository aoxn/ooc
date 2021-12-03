package operator

import (
	"github.com/aoxn/ovm/pkg/context/shared"
	"github.com/aoxn/ovm/pkg/operator/controllers/addon"
	"github.com/aoxn/ovm/pkg/operator/controllers/master"
	"github.com/aoxn/ovm/pkg/operator/controllers/nodepool"
	"github.com/aoxn/ovm/pkg/operator/controllers/noderepair"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, *shared.SharedOperatorContext) error

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, master.AddMaster)
	AddToManagerFuncs = append(AddToManagerFuncs, master.AddMasterSet)
	AddToManagerFuncs = append(AddToManagerFuncs, addon.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, master.AddNode)

	AddToManagerFuncs = append(AddToManagerFuncs, nodepool.AddNodePoolController)
	AddToManagerFuncs = append(AddToManagerFuncs, noderepair.AddNodeRepair)
}

// AddControllers adds all Controllers to the Manager
func AddControllers(
	mgr manager.Manager,
	oper *shared.SharedOperatorContext,
) error {
	for _, f := range AddToManagerFuncs {
		err := f(mgr, oper)
		if err != nil {
			return err
		}
	}
	return nil
}
