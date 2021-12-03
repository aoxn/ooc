package operator

import (
	"flag"
	"fmt"
	"github.com/docker/distribution/uuid"
	"github.com/getlantern/deepcopy"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	//"github.com/spf13/pflag"
	"github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/operator"
	"github.com/aoxn/ovm/pkg/utils"
	"io/ioutil"
	//"os"
	//ctrl "sigs.k8s.io/controller-runtime"
	//"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &v1.OperatorFlag{}
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Kubernetes cluster operator",
		Long:  "kubernetes cluster operator. configuration management, lifecycle management",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.MetricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().BoolVar(&flags.EnableLeader, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	cmd.Flags().StringVar(&flags.Token, "token", "abcd.1234567890", "authentication token")
	cmd.Flags().StringVar(&flags.BindAddr, "bind-addr", "", "bind address")
	cmd.Flags().IntVar(&flags.InitialCount, "initial-count", 3, "initial master count for bootstrap")
	cmd.Flags().StringVar(&flags.MetaConfig, "bootcfg", "", "bootstrap cluster config file")
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	return cmd
}

func runE(flags *v1.OperatorFlag, cmd *cobra.Command, args []string) error {
	opts := &v1.OvmOptions{
		OperatorCFG: *flags,
	}

	klog.Infof("try start operator at [%s]", flags.BindAddr)
	server := operator.NewOperatorServer(opts)
	err := server.Start()
	if err != nil {
		panic(fmt.Sprintf("run apiserver: %s", err.Error()))
	}
	klog.Infof("server started at [%s]", flags.BindAddr)
	defer klog.Infof("operator server shutdown automatically.")
	select {}
	return nil
}

func LoadBootCfgOrDie(filen string) *v1.ClusterSpec {
	cont, err := ioutil.ReadFile(filen)
	if err != nil {
		panic(fmt.Errorf("load cluster config error: %s", err.Error()))
	}
	spec := &v1.ClusterSpec{}
	err = yaml.Unmarshal(cont, spec)
	if err != nil {
		panic(fmt.Sprintf("bootcfg file provided,but not valid %s", err.Error()))
	}
	var output = &v1.ClusterSpec{}
	err = deepcopy.Copy(output, spec)
	if err != nil {
		panic(fmt.Sprintf("deecopy: %s", err.Error()))
	}
	output.Kubernetes.RootCA.Key = []byte("[MASK]")
	output.Kubernetes.RootCA.Cert = []byte("[MASK]")
	output.Kubernetes.FrontProxyCA.Cert = []byte("[MASK]")
	output.Kubernetes.FrontProxyCA.Key = []byte("[MASK]")
	output.Kubernetes.SvcAccountCA.Key = []byte("[MASK]")
	output.Kubernetes.SvcAccountCA.Cert = []byte("[MASK]")
	if output.Kubernetes.ControlRoot != nil {
		output.Kubernetes.ControlRoot.Cert = []byte("[MASK]")
		output.Kubernetes.ControlRoot.Key = []byte("[MASK]")
	}
	output.Etcd.PeerCA.Key = []byte("[MASK]")
	output.Etcd.PeerCA.Cert = []byte("[MASK]")
	output.Etcd.ServerCA.Cert = []byte("[MASK]")
	output.Etcd.ServerCA.Key = []byte("[MASK]")
	fmt.Printf("BOOTCNFG, ONBOOT:\n%s\n", utils.PrettyYaml(output))
	if spec.Kubernetes.RootCA == nil {
		panic("new self signed cert pair fail Kubernetes.RootCA")
	}
	if spec.Kubernetes.FrontProxyCA == nil {
		panic("new self signed cert pair fail Kubernetes.FrontProxyCA")
	}
	if spec.Kubernetes.SvcAccountCA == nil {
		panic("new self signed cert pair fail Kubernetes.SvcAccountCA")
	}
	if spec.Kubernetes.KubeadmToken == "" {
		panic(fmt.Sprintf("kubeadm token empty"))
	}
	if spec.Etcd.ServerCA == nil {
		panic("new self signed cert pair fail Etcd.ServerCA")
	}
	if spec.Etcd.PeerCA == nil {
		panic("new self signed cert pair fail Etcd.PeerCA")
	}
	if spec.Etcd.InitToken == "" {
		spec.Etcd.InitToken = uuid.Generate().String()
	}
	return spec
}
