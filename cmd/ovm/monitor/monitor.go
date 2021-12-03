package monitor

import (
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/operator/monit"
	"github.com/aoxn/ovm/pkg/operator/monit/check"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const mhelp = `
monit cluster from a remote backup
ovm --name kubernetes-ovm-64 \
	--monit-mode node
`

func NewCommand() *cobra.Command {
	flags := &api.OvmOptions{}
	cmd := &cobra.Command{
		Use:   "monit",
		Short: "monit kubernetes master",
		Long:  mhelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVarP(&flags.ClusterName, "name", "n", "", "the cluster to monit")
	return cmd
}

func runE(
	flags *api.OvmOptions,
	cmd  *cobra.Command,
	args []string,
) error {
	mon := monit.NewMonitor()
	cctl,err := monit.NewClusterCtl()
	if err != nil {
		return errors.Wrapf(err, "new cluster controller")
	}
	etcd, err := check.NewCheckEtcd(cctl.GetClient(), 0.1)
	if err != nil {
		return errors.Wrapf(err, "new etcd checker:")
	}
	mon.WithCheck(etcd)
	action := func() error {
		spec := etcd.Cluster.Spec
		ctx, err := provider.NewContext(flags, &spec)
		if err != nil {
			return errors.Wrapf(err, "initialize ovm context")
		}
		index := ctx.Indexer()
		id, err := index.Get(spec.ClusterID)
		if err != nil {
			return errors.Wrapf(err, "can not get clusterid from backup: %s", spec.ClusterID)
		}
		pvd := ctx.Provider()
		_, err = pvd.Recover(ctx, &id)
		return err
	}
	mon.WithAction(action)
	return mon.StartMonit()
}
