package operator

import (
	mctx "context"
	"encoding/json"
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/apiserver"
	"github.com/aoxn/ovm/pkg/apiserver/auth"
	sctx "github.com/aoxn/ovm/pkg/context"
	"github.com/aoxn/ovm/pkg/context/base"
	"github.com/aoxn/ovm/pkg/context/shared"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/iaas/provider/alibaba"
	"github.com/aoxn/ovm/pkg/operator/controllers/backup"
	"github.com/aoxn/ovm/pkg/operator/heal"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
	"os"

	//"github.com/aoxn/ovm/pkg/utils"
	"github.com/aoxn/ovm/pkg/utils/crd"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func NewOperatorServer(
	options *api.OvmOptions,
) *Operator {

	cfg := apiserver.Configuration{
		BindAddr: options.OperatorCFG.BindAddr,
	}
	if cfg.BindAddr == "" {
		cfg.BindAddr = ":443"
	}
	return &Operator{
		Options: options,
		Server: apiserver.Server{
			Config: cfg,
			//CachedCtx: context.NewCachedContext(boot),
			Auth: &auth.TokenAuthenticator{},
		},
	}
}

type Operator struct {
	apiserver.Server
	Initialized bool

	RestCfg  *rest.Config
	Shared   *shared.SharedOperatorContext
	Options  *api.OvmOptions
	Mgr      ctrl.Manager
	Client   client.Client
	Provider provider.Interface
}

func (v *Operator) CompleteSetting() error {
	// get client rest config
	cfg := ctrl.GetConfigOrDie()
	cfg.Insecure = true
	cfg.CAData = []byte("")
	cfg.CAFile = ""
	v.RestCfg = cfg
	// initialize cluster & master CRD resource
	err := crd.InitializeCRD(v.RestCfg)
	if err != nil {
		panic(fmt.Sprintf("register crds: %s", err.Error()))
	}
	spec, err := v.cluster(v.RestCfg)
	if err != nil {
		return fmt.Errorf("cluster: %s", err.Error())
	}
	v.CachedCtx = sctx.NewCachedContext(&spec.Spec)
	ctx, err := provider.NewContext(v.Options, &spec.Spec)
	if err != nil {
		return errors.Wrap(err, "build provider context")
	}
	v.Provider = ctx.Provider()
	// initialize ctrl.Manager, and start it
	err = v.startManager(spec)
	if err != nil {
		return fmt.Errorf("run controller: %s", err.Error())
	}
	klog.Infof("operator manager started, try to complete setting")
	v.Client = v.Mgr.GetClient()
	// Client & server.CachedCtx must be Initialized
	v.Initialized = true

	spec, err = v.cluster(v.RestCfg)
	if err != nil {
		return fmt.Errorf("cluster: %s", err.Error())
	}
	return v.initializeClusterResource(spec)
}

func (v *Operator) cluster(cfg *rest.Config) (*api.Cluster, error) {
	if v.Options.OperatorCFG.MetaConfig != "" {
		klog.Infof("Initialized from metaconfig: %s", v.Options.OperatorCFG.MetaConfig)
		return &api.Cluster{}, nil
	}
	mcfg := *cfg
	mcfg.APIPath = "/apis"
	mcfg.GroupVersion = &api.SchemeGroupVersion
	mcfg.NegotiatedSerializer = api.Codecs.WithoutConversion()

	mclient, err := rest.RESTClientFor(&mcfg)
	if err != nil {
		return nil, fmt.Errorf("make rest client: %s", err.Error())
	}
	cluster := &api.Cluster{}
	err = mclient.Get().
		Resource("clusters").
		Name("kubernetes-cluster").
		Do(mctx.TODO()).
		Into(cluster)
	if err != nil {
		return cluster, fmt.Errorf("rest client get: %s", err.Error())
	}
	//klog.Infof("Debug: %s", utils.PrettyYaml(cluster))
	return cluster, nil
}

func (v *Operator) initializeClusterResource(spec *api.Cluster) error {
	klog.Infof("start to initialize cluster resource[%s][%s]", spec.Spec.Endpoint.Intranet, spec.Spec.Bind.ResourceId)
	if spec.Spec.Endpoint.Intranet == "" ||
		spec.Spec.Bind.ResourceId == "" {
		klog.Infof("start to fix stack id and slb endpoint ")
		resource, err := v.Provider.GetStackOutPuts(
			provider.NewContextWithCluster(&spec.Spec),
			&api.ClusterId{
				ObjectMeta: metav1.ObjectMeta{Name: spec.Spec.ClusterID},
			},
		)
		if err != nil {
			return fmt.Errorf("provider: list resource fail, %s", err.Error())
		}
		//fmt.Printf(utils.PrettyYaml(resource))
		intranet, internet, err := findEndpoint(resource)
		if err != nil {
			return fmt.Errorf("find slb endpoint: %s", err.Error())
		}
		klog.Infof("intranet endpoint is empty, try update: %s/%s/%s", internet, intranet, resource[alibaba.StackID].Val.(string))
		spec.Spec.Endpoint.Intranet = intranet
		spec.Spec.Endpoint.Internet = internet
		stackid := resource[alibaba.StackID].Val.(string)
		if stackid == "" {
			panic("stack id should not be empty")
		}
		spec.Spec.Bind.ResourceId = stackid
		err = patchEndpoint(v.Client, spec)
		if err != nil {
			return fmt.Errorf("patch endpoint: %s", err.Error())
		}
		klog.Infof("stack id and endpoint fixed: [%s] [%s]", spec.Spec.Endpoint.Intranet, spec.Spec.Bind.ResourceId)
	}
	return v.RefreshNodeCache()
}

func patchEndpoint(
	kcli client.Client,
	spec *api.Cluster,
) error {
	ospec := &api.Cluster{}
	err := kcli.Get(
		mctx.TODO(),
		client.ObjectKey{
			Name: "kubernetes-cluster",
		}, ospec,
	)
	if err != nil {
		return fmt.Errorf("load bootcfg from apiserver: %s", err.Error())
	}
	nspec := ospec.DeepCopy()
	nspec.Spec.Endpoint.Intranet = spec.Spec.Endpoint.Intranet
	nspec.Spec.Endpoint.Internet = spec.Spec.Endpoint.Internet
	nspec.Spec.Bind.ResourceId = spec.Spec.Bind.ResourceId

	oldData, err := json.Marshal(ospec)
	if err != nil {
		return fmt.Errorf("marshal ospec: %s", err.Error())
	}
	newData, err := json.Marshal(nspec)
	if err != nil {
		return fmt.Errorf("marshal nspec: %s", err.Error())
	}
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, nspec)
	if patchErr != nil {
		return fmt.Errorf("create merge patch: %s", patchErr.Error())
	}
	return kcli.Patch(
		mctx.TODO(), nspec,
		client.RawPatch(types.MergePatchType, patchBytes),
	)
}

func findEndpoint(
	resource map[string]provider.Value,
) (string, string, error) {
	// intranet, internet
	intranet, ok := resource["APIServerIntranet"]
	if !ok {
		return "", "", fmt.Errorf("intranet slb ip not found")
	}
	internet := fmt.Sprintf("%s", resource["APIServerInternet"].Val)
	return fmt.Sprintf("%s", intranet.Val), internet, nil
}

func (v *Operator) Start() error {
	if v.Initialized {
		return fmt.Errorf("operator server has already started")
	}
	err := v.CompleteSetting()
	if err != nil {
		return err
	}
	return v.Server.Start()
}

func (v *Operator) startManager(spec *api.Cluster) error {
	speriod := 90 * time.Second
	mgr, err := ctrl.NewManager(
		v.RestCfg,
		ctrl.Options{
			SyncPeriod:              &speriod,
			Scheme:                  api.Scheme,
			MetricsBindAddress:      ":8888",
			Port:                    9443,
			LeaderElection:          true,
			LeaderElectionID:        "ovm.alibabacloud.com",
			LeaderElectionNamespace: "kube-system",
		},
	)
	v.Mgr = mgr
	if err != nil {
		panic(fmt.Sprintf("unable to start manager: %s", err.Error()))
	}
	// add schema
	err = api.AddToScheme(mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("add api schema: %s", err.Error())
	}
	err = corev1.AddToScheme(mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("add core api to schema: %s", err.Error())
	}

	mclient, err := kubernetes.NewForConfig(mgr.GetConfig())
	drainer := &drain.Helper{
		Timeout:             15 * time.Minute,
		Client:              mclient,
		GracePeriodSeconds:  -1,
		DisableEviction:     false,
		IgnoreAllDaemonSets: true,
		Force:               true,
		Out:                 os.Stdout,
		ErrOut:              os.Stderr,

		SkipWaitForDeleteTimeoutSeconds: 60,
	}
	// start member heal
	mh, err := heal.NewHealet(spec,v.Mgr.GetClient(), v.Provider, drainer)
	if err != nil {
		return errors.Wrap(err, "master heal")
	}
	err = mgr.Add(mh)
	if err != nil {
		klog.Errorf("add Healet runner: %s", err.Error())
	}
	err = mgr.Add(backup.NewSnapshot())
	if err != nil {
		klog.Errorf("add Snapshot runner: %s", err.Error())
	}

	pctx, err := LoadContextIAAS(v.Provider, spec)
	if err != nil {
		return fmt.Errorf("provider context: %s", err.Error())
	}
	pctx.SetKV("Provider", v.Provider)
	v.Shared = shared.NewOperatorContext(v.CachedCtx, v.Provider, mh, pctx)

	// add controllers
	err = AddControllers(mgr, v.Shared)
	if err != nil {
		return fmt.Errorf("add controllers: %s", err.Error())
	}
	start := func() {
		klog.Infof("starting manager for: %+v", mgr.GetConfig().Host)
		err = mgr.Start(context.TODO())
		if err != nil {
			panic(fmt.Sprintf("run controller manager: %s", err.Error()))
		}
		panic("controller manager stopped")
	}
	go start()
	klog.Infof("wait for lease, and wait for cache sync")
	// wait for manager cache sync
	if !mgr.GetCache().WaitForCacheSync(context.TODO()) {
		return fmt.Errorf("wait for manager sync")
	}
	klog.Infof("manager cache synced")
	return nil
}

func (v *Operator) RefreshNodeCache() error {
	list := &api.MasterList{}
	err := v.Client.List(mctx.TODO(), list)
	if err != nil {
		return fmt.Errorf("build node cache: %s", err.Error())
	}
	for _, node := range list.Items {
		// cache master for further use
		v.CachedCtx.AddMaster(node)
	}
	logcache(v.CachedCtx.Nodes)
	return nil
}

func LoadContextIAAS(
	prvd provider.Interface,
	spec *api.Cluster,
) (*provider.Context, error) {
	ctx := provider.NewContextWithCluster(&spec.Spec)
	if spec.Spec.Bind.ResourceId == "" {
		resource, err := prvd.GetStackOutPuts(
			ctx,
			&api.ClusterId{
				ObjectMeta: metav1.ObjectMeta{
					Name: spec.Spec.ClusterID,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("provider: list resource fail, %s", err.Error())
		}
		spec.Spec.Bind.ResourceId = resource[alibaba.StackID].Val.(string)
	}
	stack, err := prvd.GetInfraStack(
		ctx, &api.ClusterId{Spec: api.ClusterIdSpec{ResourceId: spec.Spec.Bind.ResourceId}},
	)
	if err != nil {
		return ctx, err
	}
	return ctx.WithStack(stack), nil
}

func logcache(ctx *base.Context) {
	ctx.Range(
		func(key, value interface{}) bool {
			k := key.(string)
			v := value.(api.Master)
			klog.Infof("==============================================================")
			klog.Infof("  key: %s", k)
			klog.Infof("value: %s", v.String())
			klog.Info()
			return true
		},
	)
	klog.Infof("log cache finished.")
}
