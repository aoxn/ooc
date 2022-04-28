package monit

import (
	mctx "context"
	"fmt"
	api "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	"github.com/pkg/errors"
	app "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	b "sigs.k8s.io/controller-runtime/pkg/cluster"
	"time"
)

func NewClusterCtl() (b.Cluster, error) {
	syncPeriod := 3 * time.Minute
	option := func(options *b.Options) {
		options.SyncPeriod = &syncPeriod
		options.Scheme = api.Scheme
	}
	cluster, err := b.New(config.GetConfigOrDie(), option)
	if err != nil {
		return cluster, errors.Wrapf(err, "new cluster controller")
	}
	klog.Infof("new cluster controller: add schema v1/api")
	_ = v1.AddToScheme(cluster.GetScheme())
	_ = api.AddToScheme(cluster.GetScheme())
	_ = app.AddToScheme(cluster.GetScheme())
	start := func() {
		err = cluster.Start(mctx.TODO())
		if err != nil {
			panic(fmt.Sprintf("start cache", err.Error()))
		}
	}
	go start()

	if !cluster.GetCache().WaitForCacheSync(mctx.TODO()) {
		return cluster, errors.Wrapf(err, "wait cache sync:")
	}
	klog.Infof("cache sync finished")
	return cluster, nil
}

func GetKubernetesClient(c b.Cluster) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(c.GetConfig())
}

func GetSpec(
	client client.Client,
) (*api.Cluster, []api.Master, error) {
	spec, err := h.Cluster(client, "kubernetes-cluster")
	if err != nil {
		return nil, nil, err
	}
	masters, err := h.MasterCRDS(client)
	if err != nil {
		return nil, nil, err
	}
	return spec, masters, nil
}

func NewRest(cfg *rest.Config) (*rest.RESTClient, error) {
	cfg.APIPath = "/apis"
	cfg.GroupVersion = &api.SchemeGroupVersion
	cfg.NegotiatedSerializer = api.Codecs.WithoutConversion()

	return rest.RESTClientFor(cfg)
}

var last_chaos_time = "last.chaos.time"

func LoadLastChaosTime(
	mclient client.Client,
) (time.Time, error) {
	cm := v1.ConfigMap{}
	err := mclient.Get(
		mctx.TODO(),
		client.ObjectKey{Namespace: "kube-system", Name: "chaos.config"}, &cm,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return time.Now().Add(-24 * time.Hour), nil
		}
		return time.Time{}, err
	}
	return time.Parse("2006-01-02 15:04:05", cm.Data[last_chaos_time])
}

func SaveLastChaosTime(
	mclient client.Client, now time.Time,
) error {
	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chaos.config",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			last_chaos_time: now.Format("2006-01-02 15:04:05"),
		},
	}
	err := mclient.Create(mctx.TODO(), &cm)
	if err != nil {
		return mclient.Update(mctx.TODO(), &cm)
	}
	return nil
}
