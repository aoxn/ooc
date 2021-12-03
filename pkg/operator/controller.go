package operator

import (
	"github.com/aoxn/ooc/pkg/context/shared"
	"github.com/aoxn/ooc/pkg/operator/controllers/addon"
	"github.com/aoxn/ooc/pkg/operator/controllers/autorepair"
	"github.com/aoxn/ooc/pkg/operator/controllers/master"
	"github.com/aoxn/ooc/pkg/operator/controllers/nodepool"
	"github.com/aoxn/ooc/pkg/operator/controllers/rolling"
	"github.com/aoxn/ooc/pkg/operator/controllers/task"
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
	AddToManagerFuncs = append(AddToManagerFuncs, task.AddTaskController)
	AddToManagerFuncs = append(AddToManagerFuncs, rolling.AddRollingController)

	AddToManagerFuncs = append(AddToManagerFuncs, nodepool.AddNodePoolController)
	AddToManagerFuncs = append(AddToManagerFuncs, autorepair.AddAutoRepairController)
}

// AddControllers adds all Controllers to the Manager
func AddControllers(
	mgr  manager.Manager,
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
