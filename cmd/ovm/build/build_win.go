//go:build windows
// +build windows

package build

import (
	"fmt"
	"github.com/spf13/cobra"
)

type flagpole struct {
	DryRun bool
	Cache  bool
	OS     string
	Arch   string
	// Beta prerelease
	NamespaceFrom        string
	NamespaceTo          string
	BaseFileServer       string
	TempDir              string
	KubernetesVersion    string
	KubernetesCNIVersion string
	DockerVersion        string
	EtcdVersion      string
	OvmVersion       string
	RunScriptVersion string
	CloudType            string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	build := &cobra.Command{
		Use:   "build",
		Short: "Kubernetes cluster build package",
		Long: `kubernetes cluster build package.
	runtime/kubernetes/etcd/kuberentes-cni/ovm
	from: ovm/${BETA}/${VERSION}/ovm/amd64/linux/files
	to:   
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	build.Flags().BoolVar(&flags.DryRun, "dry-run", false, "dry run")
	build.Flags().BoolVar(&flags.Cache, "cache", false, "cache downloaded file")
	build.Flags().StringVar(&flags.OS, "os", "linux", "os type default linux")
	build.Flags().StringVar(&flags.Arch, "arch", "amd64", "arch ,default amd64")
	build.Flags().StringVar(&flags.NamespaceFrom, "namespace-from", "aoxn", "default to aoxn")
	build.Flags().StringVar(&flags.NamespaceTo, "namespace-to", "default", "default to default")
	build.Flags().StringVar(&flags.DockerVersion, "runtime-version", "18.09.2", "runtime version ")
	build.Flags().StringVar(&flags.KubernetesVersion, "kubernetes-version", "1.12.6-aliyun.1", "kubernetes version 1.12.6-aliyun.1")
	build.Flags().StringVar(&flags.KubernetesCNIVersion, "kubernetes-cni-version", "0.5.1", "kubernetes cni version 0.6.0")
	build.Flags().StringVar(&flags.CloudType, "cloud-type", "public", "cloud type default public ")
	build.Flags().StringVar(&flags.BaseFileServer, "download-from", "", "download pkg from which server")
	build.Flags().StringVar(&flags.EtcdVersion, "etcd-version", "v3.3.8", "etcd version ")
	build.Flags().StringVar(&flags.OvmVersion, "ovm-version", "0.1.0", "ovm version ")
	build.Flags().StringVar(&flags.RunScriptVersion, "run-script-version", "2.0", "run script version ")
	return build
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {

	return fmt.Errorf("not implemented")
}
