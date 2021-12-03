package monkey

import (
	"fmt"
	v1 "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	prvd "github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/iaas/provider/alibaba"
	"github.com/aoxn/ovm/pkg/operator/controllers/backup"
	"github.com/aoxn/ovm/pkg/operator/monit"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"time"
)

func NewOperations(ctx *prvd.Context) []Action{
	bootcfg := ctx.BootCFG()
	return []Action{
		NewDeleteECS(ctx, bootcfg.ClusterID),
		NewStopECS(ctx, bootcfg.ClusterID),
		NewStopProcess(ctx, bootcfg.ClusterID, "docker"),
		NewStopProcess(ctx, bootcfg.ClusterID, "etcd"),
		NewStopProcess(ctx, bootcfg.ClusterID, "kubelet"),
	}
}

func printx(o []Action) {
	for k, v := range o {
		fmt.Printf("%d: %s\n", k, v.Name())
	}
}

func TriggerBackup(ctx *prvd.Context, client client.Client) {
	spec, masters, err := monit.GetSpec(client)
	if err != nil {
		panic(fmt.Sprintf("init master: %s", err.Error()))
	}
	snap := backup.NewBareSnapshot(ctx.Indexer())
	err = snap.Backup(spec, masters)
	if err != nil {
		panic(fmt.Sprintf("backup failed: %s", err.Error()))
	}
	klog.Infof("backup etcd finished with no surprise")
}

func ChaosMonkeyJump() {
	klog.Infof("run chaos monkey")
	klog.Infof("trigger etcd backup first...")
	// trigger backup first
	restc, ctx, err := NewInitializedPrvdCtx()
	if err != nil {
		panic(fmt.Sprintf("initialize provider context:%s", err.Error()))
	}

	jump := func() {
		last, err := monit.LoadLastChaosTime(restc.GetClient())
		if err != nil {
			panic(fmt.Sprintf("last chaos config: %s", err.Error()))
		}
		klog.Infof("last chaos time: [%s], [%t]",
			last.Format("2006-01-02 15:04:05"),time.Now().After(last.Add(1*time.Hour)),
		)
		if !time.Now().After(last.Add(1*time.Hour)) {
			return
		}
		klog.Infof("monkey jump begin...............................")
		Jump(ctx, restc.GetClient())
	}
	wait.Forever(jump, 1 * time.Minute)
}

func Jump(ctx *prvd.Context, client client.Client) {
	err := monit.SaveLastChaosTime(client,time.Now())
	if err != nil {
		klog.Warningf("unable to save " +
			"last chaos time,skip chaos monkey: %s", err.Error())
		return
	}
	// trigger backup after LastChaosTime has been updated.
	TriggerBackup(ctx, client)
	operations := NewOperations(ctx)
	// at least 2 operations
	n := rand.Intn(len(operations) - 2) + 2
	klog.Infof("random pick up %d/%d [OPERATIONS]", n, len(operations))
	rand.Shuffle(
		len(operations),
		func(i, j int) {
			operations[i], operations[j] = operations[j], operations[i]
		},
	)
	total := 0
	for i := 0; i < n; i++ {
		klog.Infof("(%d). RUN OPERATION[%s]",i, operations[i].Name())
		cnt, err := operations[i].Execute()
		if err != nil {
			klog.Errorf("monkey jump fail: %s, continue", err.Error())
		}
		total += cnt
	}
	klog.Infof("monkey jump: total %d operation executed", total)
}

type Action interface {
	Name() string
	Execute() (int, error)
}

func NewInitializedPrvdCtx() (cluster.Cluster, *prvd.Context, error) {
	restc, err := monit.NewClusterCtl()
	if err != nil {
		return nil,nil, errors.Wrapf(err, "get cluster spec")
	}

	spec, _, err := monit.GetSpec(restc.GetClient())
	if err != nil {
		return nil, nil, err
	}
	mctx, err := prvd.NewContext(&v1.OvmOptions{}, &spec.Spec)
	if err != nil {
		return nil, mctx, errors.Wrapf(err, "new prvd context")
	}
	mprvd := mctx.Provider()
	index := mctx.Indexer()
	id, err := index.Get(spec.Spec.ClusterID)
	if err != nil {
		return nil, mctx, errors.Wrapf(err, "get cluster id")
	}
	stack, err := mprvd.GetInfraStack(mctx, &id)
	if err != nil {
		return nil, mctx, errors.Wrapf(err, "get stack: ")
	}
	mctx.WithStack(stack)
	return restc,mctx, nil
}

type Base struct {
	ClusterName string
	PrvdCtx     *prvd.Context
}

func (d *Base) Name() string { return "base" }

func (d *Base) doOperate(
	mfunc func(i prvd.Instance) error,
) (int, error) {
	if d.ClusterName == "" {
		return 0, fmt.Errorf("empty cluster name")
	}
	mprvd := d.PrvdCtx.Provider()
	detail, err := mprvd.ScalingGroupDetail(
		d.PrvdCtx, "", prvd.Option{Action: alibaba.ActionInstanceIDS},
	)
	if err != nil {
		return 0, errors.Wrapf(err, "find ecs for master fail: %s", err.Error())
	}

	n := rand.Intn(len(detail.Instances))/2 + 1
	if n > len(detail.Instances){
		n = len(detail.Instances)
	}
	klog.Infof("random pickup %d/%d [INSTANCE]", n,len(detail.Instances))
	k := 0
	for _, v := range detail.Instances {
		if k >= n {
			break
		}
		k++
		err := mfunc(v)
		if err != nil {
			klog.Errorf("try action failed, sleep 10s: %s", err.Error())
			time.Sleep(10 * time.Second)
		}
	}
	return n, nil
}

func (d *Base) Execute() (int, error) { return 0, nil }

func NewDeleteECS(
	ctx *prvd.Context, name string,
) *DeleteECS {
	action := DeleteECS{
		Base: Base{
			PrvdCtx:     ctx,
			ClusterName: name,
		},
	}
	return &action
}

type DeleteECS struct{ Base }

func (d *DeleteECS) Name() string { return "delete.ecs" }

func (d *DeleteECS) Execute() (int, error) {
	mprvd := d.PrvdCtx.Provider()
	mfunc := func(i prvd.Instance) error {
		klog.Infof("[Action] DeleteECS: %s", i.Id)
		return mprvd.DeleteECS(d.PrvdCtx, i.Id)
	}
	return d.doOperate(mfunc)
}

func NewStopECS(
	ctx *prvd.Context, name string,
) *StopECS {
	return &StopECS{
		Base: Base{
			PrvdCtx:     ctx,
			ClusterName: name,
		},
	}
}

type StopECS struct{ Base }

func (d *StopECS) Name() string { return "stop.ecs" }

func (d *StopECS) Execute() (int, error) {
	mprvd := d.PrvdCtx.Provider()
	mfunc := func(i prvd.Instance) error {
		klog.Infof("[Action] StopECS: %s", i.Id)
		return mprvd.StopECS(d.PrvdCtx, i.Id)
	}
	return d.doOperate(mfunc)
}

func NewStopProcess(
	ctx *prvd.Context, name, process string,
) *StopProcess {
	return &StopProcess{
		Base: Base{
			PrvdCtx:     ctx,
			ClusterName: name,
		},
		Process: process,
	}
}

type StopProcess struct {
	Base
	Process string
}

func (d *StopProcess) Name() string { return fmt.Sprintf("stop.%s", d.Process) }

func (d *StopProcess) Execute() (int, error) {
	cmd := fmt.Sprintf("systemctl stop %s", d.Process)
	mprvd := d.PrvdCtx.Provider()
	mfunc := func(i prvd.Instance) error {
		klog.Infof("[Action] RunCommand: %s, %s", i.Id, cmd)
		return mprvd.RunCommand(d.PrvdCtx, i.Id, cmd)
	}
	return d.doOperate(mfunc)
}
