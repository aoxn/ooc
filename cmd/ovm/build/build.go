//go:build linux || darwin
// +build linux darwin

package build

import (
	"fmt"
	"github.com/aoxn/ovm/pkg/actions/file"
	"github.com/aoxn/ovm/pkg/utils"
	"github.com/aoxn/ovm/pkg/utils/cmd"
	"github.com/getlantern/deepcopy"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

/*
	二进制下载地址：
		cni:   https://github.com/containernetworking/plugins/releases/
        docker: https://download.docker.com/linux/static/stable/x86_64/

*/

type flagpole struct {
	// comma separated
	Mode     string
	Regions  []string
	Bucket   string
	Download bool
	DryRun   bool
	Cache    bool
	OS       string
	SourceOS string
	Arch     string

	Project string
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
	from: ovm/${NAMESPACE}/${VERSION}/ovm/amd64/linux/files
	to:   
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	build.Flags().StringVar(&flags.Bucket, "bucket", "host-ovm", "default bucket prefix. real bucket name would be ${prefix}-${region}, eg. host-ovm-cn-hangzhou")
	build.Flags().BoolVar(&flags.Download, "download", false, "download cache from remote default server")
	build.Flags().BoolVar(&flags.DryRun, "dry-run", false, "dry run")
	build.Flags().StringVar(&flags.Mode, "mode", "local", "pkg from local or remote")
	build.Flags().BoolVar(&flags.Cache, "cache", false, "cache downloaded file")
	build.Flags().StringVar(&flags.SourceOS, "source-os", "centos", "os type default centos")
	build.Flags().StringVar(&flags.Project, "project", "ovm", "project type default ovm")
	build.Flags().StringVar(&flags.OS, "os", "centos", "os type default centos")
	build.Flags().StringVar(&flags.Arch, "arch", "amd64", "arch ,default amd64")
	build.Flags().StringArrayVar(&flags.Regions, "regions", []string{"cn-hangzhou"}, "arch ,default cn-hangzhou")
	build.Flags().StringVar(&flags.NamespaceFrom, "namespace-from", "default", "default to aoxn")
	build.Flags().StringVar(&flags.NamespaceTo, "namespace-to", "default", "default to default")
	build.Flags().StringVar(&flags.DockerVersion, "runtime-version", "", "runtime version 19.03.5")
	build.Flags().StringVar(&flags.KubernetesVersion, "kubernetes-version", "", "kubernetes version 1.16.9-aliyun.1")
	build.Flags().StringVar(&flags.KubernetesCNIVersion, "kubernetes-cni-version", "", "kubernetes cni version 0.8.6")
	build.Flags().StringVar(&flags.CloudType, "cloud-type", "public", "cloud type default public ")
	build.Flags().StringVar(&flags.BaseFileServer, "download-from", "", "download pkg from which server")
	build.Flags().StringVar(&flags.EtcdVersion, "etcd-version", "", "etcd version v3.4.3")
	build.Flags().StringVar(&flags.OvmVersion, "ovm-version", "", "ovm version 0.1.1")
	build.Flags().StringVar(&flags.RunScriptVersion, "run-version", "", "run script version 2.0")
	return build
}

func runE(
	flags *flagpole,
	cmd *cobra.Command,
	args []string,
) error {
	return InitBuild(flags)
}

func InitBuild(flag *flagpole) error {
	// source: ${PROJECT}/${PKG}/${ARCH}/${OS}/bin/
	// source: ovm/kubernetes/amd64/linux/bin/{kubelet,kubectl,kubeadm}

	ctx := NewBuildContext(
		func(bctx *BuildContext) {
			bctx.Ctx = flag

			bctx.TempDir = fmt.Sprintf("%s/.ovm/cache", os.Getenv("HOME"))
		},
	)
	return ctx.Build()
}

// Option is BuildContext configuration option supplied to NewBuildContext
type Option func(*BuildContext)

func NewBuildContext(options ...Option) *BuildContext {
	ctx := &BuildContext{}
	for _, opt := range options {
		opt(ctx)
	}
	return ctx
}

type BuildContext struct {
	Ctx     *flagpole
	TempDir string
}

type build struct {
	pkg *file.Path
	run func(b *BuildContext, m *build) error
}

func (b *build) download(f *flagpole) error {
	base := fmt.Sprintf("http://%s-%s.oss.aliyuncs.com", f.Bucket, f.Regions[0])
	src := fmt.Sprintf("%s/%s", base, b.pkg.URI())
	if err := os.MkdirAll(b.pkg.URL(), 0755); err != nil {
		return fmt.Errorf("enusre dire %s : %s", b.pkg.URL(), err.Error())
	}
	cm := cmd.NewCmd(
		"wget", src, "-O", b.pkg.URI(),
	)
	result := <-cm.Start()

	return cmd.CmdError(result)
}

func (b *build) mkExampleDir(home string) error {
	path := filepath.Join(home, b.pkg.URL_T())
	klog.Infof("make dir: %s", path)
	return os.MkdirAll(path, 0755)
}

func (b *BuildContext) info() string {
	info := []string{
		fmt.Sprintf("bucket[%s]",b.Ctx.Bucket),
		fmt.Sprintf("arch[%s]", b.Ctx.Arch),
		fmt.Sprintf("region[%s]", b.Ctx.Regions),
		fmt.Sprintf("os[%s]", b.Ctx.OS),
		fmt.Sprintf("cloud-type[%s]", b.Ctx.CloudType),
	}
	return strings.Join(info, ",")
}

func (b *BuildContext) Build() error {

	klog.Infof("build context: %s", b.info())

	if b.Ctx.DryRun {
		klog.Infof("[dry-run] mode")
		if len(b.Ctx.Regions) < 1 {
			klog.Errorf("regions must be provided")
			os.Exit(1)
		}
		bucket := fmt.Sprintf("%s-%s", b.Ctx.Bucket, b.Ctx.Regions[0])
		klog.Infof("oss://%s", bucket)
		klog.Infof("http://%s.oss.aliyuncs.com"+
			"/ovm/default/public/etcd/v3.4.3/amd64/centos/bin/etcd-v3.4.3.tar.gz", bucket)
	}
	for k, v := range []build{
		{pkg: NewDocker(b), run: BuildDocker},
		{pkg: NewEtcd(b), run: BuildEtcd},
		{pkg: NewKubernetes(b), run: BuildKubernetes},
		{pkg: NewCNI(b), run: BuildCNI},
		{pkg: NewRun(b), run: BuildRunScript},
		{pkg: NewOvm(b), run: BuildOvm},
	} {

		if b.Ctx.DryRun {
			err := v.mkExampleDir(b.TempDir)
			if err != nil {
				return fmt.Errorf("mkdir example dir failed")
			}
			continue
		}

		if v.pkg.Version == "" {
			// no package specified, skip build
			continue
		}

		if b.Ctx.Download {
			if err := v.download(b.Ctx); err != nil {
				return fmt.Errorf("download: %s", err.Error())
			}
		}
		klog.Infof("[%d] start to build package[%s], version[%s] ...", k, v.pkg.Pkg, v.pkg.Version)
		if err := v.run(b, &v); err != nil {
			return fmt.Errorf("build package with error: %s", err.Error())
		}
		klog.Infof("[%d] build package[%s], version[%s] finished\n\n", k, v.pkg.Pkg, v.pkg.Version)
	}

	fmt.Printf("finish build\n")
	return nil
}

func NewEtcd(b *BuildContext) *file.Path {
	return &file.Path{
		Project:   b.Ctx.Project,
		Namespace: b.Ctx.NamespaceFrom,
		CType:     b.Ctx.CloudType,
		Pkg:       "etcd",
		Version:   b.Ctx.EtcdVersion,
		Arch:      b.Ctx.Arch,
		OS:        b.Ctx.SourceOS,
		Ftype:     file.FILE_BINARY,
	}
}

func NewKubernetes(b *BuildContext) *file.Path {
	return &file.Path{
		Project:   b.Ctx.Project,
		Namespace: b.Ctx.NamespaceFrom,
		CType:     b.Ctx.CloudType,
		Pkg:       "kubernetes",
		Version:   b.Ctx.KubernetesVersion,
		Arch:      b.Ctx.Arch,
		OS:        b.Ctx.SourceOS,
		Ftype:     file.FILE_BINARY,
	}
}

func NewDocker(b *BuildContext) *file.Path {
	return &file.Path{
		Project:   b.Ctx.Project,
		Namespace: b.Ctx.NamespaceFrom,
		CType:     b.Ctx.CloudType,
		Pkg:       "docker",
		Version:   b.Ctx.DockerVersion,
		Arch:      b.Ctx.Arch,
		OS:        b.Ctx.SourceOS,
		Ftype:     file.FILE_BINARY,
	}
}

func NewOvm(b *BuildContext) *file.Path {
	return &file.Path{
		Project:   b.Ctx.Project,
		Namespace: b.Ctx.NamespaceFrom,
		CType:     b.Ctx.CloudType,
		Pkg:       "ovm",
		Version:   b.Ctx.OvmVersion,
		Arch:      b.Ctx.Arch,
		OS:        b.Ctx.SourceOS,
		Ftype:     file.FILE_BINARY,
	}
}

func NewRun(b *BuildContext) *file.Path {
	return &file.Path{
		Project:   b.Ctx.Project,
		Namespace: b.Ctx.NamespaceFrom,
		CType:     b.Ctx.CloudType,
		Pkg:       "run",
		Version:   b.Ctx.RunScriptVersion,
		Arch:      b.Ctx.Arch,
		OS:        b.Ctx.SourceOS,
		Ftype:     file.FILE_BINARY,
	}
}

func NewCNI(b *BuildContext) *file.Path {
	return &file.Path{
		Project:   b.Ctx.Project,
		Namespace: b.Ctx.NamespaceFrom,
		CType:     b.Ctx.CloudType,
		Pkg:       "kubernetes-cni",
		Version:   b.Ctx.KubernetesCNIVersion,
		Arch:      b.Ctx.Arch,
		OS:        b.Ctx.SourceOS,
		Ftype:     file.FILE_BINARY,
	}
}

func BuildCNI(b *BuildContext, m *build) error { return doBuild(b, m.pkg) }

func BuildDocker(b *BuildContext, m *build) error { return doBuild(b, m.pkg) }

func BuildEtcd(b *BuildContext, m *build) error { return doBuild(b, m.pkg) }

func BuildKubernetes(b *BuildContext, m *build) error { return doBuild(b, m.pkg) }

func BuildOvm(b *BuildContext, m *build) error { return doBuildOvm(b, m.pkg) }

func BuildRunScript(b *BuildContext, m *build) error {

	for _, pro := range []string{"alibaba", "boot", "replace"} {
		if err := copyRunScript(b, m.pkg, pro); err != nil {
			return fmt.Errorf("build run script fail: %s", err.Error())
		}
	}
	return nil
}

func doBuildOvm(b *BuildContext, from *file.Path) error {
	bcmd := "ovmmac"
	target := filepath.Join(fmt.Sprintf("build/bin/ovm"))
	switch b.Ctx.Arch {
	case "arm64":
		bcmd = "ovmarm64"
		target = filepath.Join(fmt.Sprintf("build/bin/ovm.arm64"))
	case "amd64":
		bcmd = "ovmlinux"
		target = filepath.Join(fmt.Sprintf("build/bin/ovm.amd64"))
	}
	m := cmd.NewCmd(
		"make", bcmd,
	)
	status := <-m.Start()
	if err := cmd.CmdError(status); err != nil {
		return fmt.Errorf("build ovm error: %s", err.Error())
	}

	to := file.Path{}
	err := deepcopy.Copy(&to, &from)
	if err != nil {
		return err
	}
	to.OS = b.Ctx.OS
	to.Namespace = b.Ctx.NamespaceTo
	f := file.Transfer{
		Bucket:  b.Ctx.Bucket,
		Regions: b.Ctx.Regions,
		Upload:  Upload2OSS,
		From:    from,
		To:      &to,
		Cache:   b.TempDir,
		Base:    b.Ctx.BaseFileServer,
	}

	return f.Upload(&f, target, filepath.Join(f.To.URL()))
}

func copyRunScript(b *BuildContext, from *file.Path, provider string) error {
	to := file.Path{}
	err := deepcopy.Copy(&to, &from)
	if err != nil {
		return err
	}
	to.OS = b.Ctx.OS
	to.Namespace = b.Ctx.NamespaceTo
	f := file.Transfer{
		Bucket:  b.Ctx.Bucket,
		Regions: b.Ctx.Regions,
		Upload:  Upload2OSS,
		From:    from,
		To:      &to,
		Cache:   b.TempDir,
		Base:    b.Ctx.BaseFileServer,
	}

	name := fmt.Sprintf("run.%s.sh", provider)

	return f.Upload(&f, fmt.Sprintf("run/%s", name), f.To.URL())
}

func doBuild(
	b *BuildContext,
	from *file.Path,
) error {

	to := &file.Path{}
	err := deepcopy.Copy(to, from)
	if err != nil {
		return err
	}
	to.OS = b.Ctx.OS
	to.Namespace = b.Ctx.NamespaceTo
	tar := NewTar(
		filepath.Join(b.TempDir, from.URL()),
		filepath.Join(b.TempDir, from.Name()),
	)
	f := file.Transfer{
		Bucket:   b.Ctx.Bucket,
		Regions:  b.Ctx.Regions,
		Tar:      tar,
		Upload:   Upload2OSS,
		Download: DownloadFromOSS,
		From:     from,
		To:       to,
		Cache:    b.TempDir,
		Base:     b.Ctx.BaseFileServer,
	}
	//switch b.Ctx.Mode {
	//case MODE_LOCAL:
	//	// default value
	//case MODE_REMOTE:
	//	//
	//	cache := deepcopy.Copy(from)
	//	err := f.Download(&f, f.From.URI(), f.To.URL())
	//	if err != nil {
	//		return err
	//	}
	//}

	err = f.Tar.Tar()
	if err != nil {
		return err
	}
	return f.Upload(&f, f.Tar.Location(), to.URL())
}

func NewTar(from, name string) *Tar {
	return &Tar{from: from, name: name}
}

type Tar struct {
	from string
	name string
}

func (m *Tar) Tar() error {
	exist, err := utils.FileExist(m.from)
	if err != nil || !exist {
		return fmt.Errorf("file not exist or read file error: %v", err)
	}
	gcm := cmd.NewCmd(
		"tar", "zcvf", m.name, "-C", m.from, ".",
	)
	gcm.Env = append(gcm.Env, os.Environ()...)
	// tell mac tar to skip hidden files (etc. ._yaml.txt)
	gcm.Env = append(gcm.Env, "COPYFILE_DISABLE=1")
	sta := <-gcm.Start()
	return cmd.CmdError(sta)
}

func (m *Tar) Location() string { return m.name }

// DownloadFromOSS
// download from f.From.URI --> f.To.URL
func DownloadFromOSS(
	f *file.Transfer, from, to string,
) error {
	err := os.MkdirAll(to, 0755)
	if err != nil {
		return fmt.Errorf("download: [mkdir -p] %s", err.Error())
	}
	if len(f.Regions) <= 0 {
		return fmt.Errorf("region not specified, [%s]", f.Regions)
	}
	endpoint := fmt.Sprintf("%s.oss.aliyuncs.com", f.Regions[0])
	m := cmd.NewCmd(
		"ossutil", "--endpoint", endpoint, "cp", "-r",
		fmt.Sprintf("oss://%s/%s/", f.Bucket, from), to,
	)
	status := <-m.Start()
	return cmd.CmdError(status)
}

// Upload2OSS
// upload from f.From.URI --> f.To.URL
func Upload2OSS(f *file.Transfer, from, to string) error {
	klog.Infof("upload region: %s", f.Regions)
	for _, region := range f.Regions {
		endpoint := fmt.Sprintf("%s.oss.aliyuncs.com", region)
		sta := <-cmd.NewCmd(
			"ossutil", "--endpoint", endpoint, "cp", "-u",
			from, fmt.Sprintf("oss://%s-%s/%s/", f.Bucket, region, to),
		).Start()
		if err := cmd.CmdError(sta); err != nil {
			return fmt.Errorf("upload pkg: %s", err.Error())
		}
	}
	return nil
}
