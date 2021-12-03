package file

import (
	"fmt"
	"github.com/aoxn/ovm/pkg/utils"
	"github.com/aoxn/ovm/pkg/utils/cmd"
	tar "github.com/verybluebot/tarinator-go"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	FILE_BINARY  = "bin"
	FILE_RPM     = "rpm"
	FILE_REGULAR = "regular"
)

const (
	PKG_DOCKER     = "docker"
	PKG_KUBERNETES = "kubernetes"
	PKG_CNI        = "kubernetes-cni"
	PKG_ETCD       = "etcd"
)

func wget(f *File) string {
	if f.BaseServer == "" {
		f.BaseServer = fmt.Sprintf("http://%s-%s.oss-%s-internal.aliyuncs.com/", f.Bucket, f.Region, f.Region)
	}
	return fmt.Sprintf("%s/%s", f.BaseServer, f.VersionedPath.URI())
}

func (f *File) cacheDir() string {
	return filepath.Join(f.CacheDir, "cache", f.VersionedPath.Pkg)
}

func (f *File) extractDir() string {
	return filepath.Join(f.CacheDir, "extract", f.VersionedPath.Pkg)
}

func withName(path, name string) string {
	return filepath.Join(path, name)
}

type File struct {
	Bucket        string
	Region        string
	BaseServer    string
	CacheDir      string
	VersionedPath Path
}

func (f *File) Download() error {
	if err := os.MkdirAll(f.cacheDir(), 0755); err != nil {
		return fmt.Errorf("enusre dire %s : %s", f.cacheDir(), err.Error())
	}
	switch f.VersionedPath.OS {
	case "centos":
		cm := cmd.NewCmd(
			"wget", "--tries", "10", "--no-check-certificate", "-q",
			wget(f),
			"-O", withName(f.cacheDir(), f.VersionedPath.Name()),
		)
		result := <-cm.Start()

		return cmd.CmdError(result)
	}

	return nil
}

func (f *File) Tar() error {
	return tar.Tarinate([]string{}, "")
}

func (f *File) Untar() error {
	return tar.UnTarinate(f.extractDir(), withName(f.cacheDir(), f.VersionedPath.Name()))
}

func (f *File) Install() error {
	errs := utils.Errors{}
	switch f.VersionedPath.Ftype {
	case FILE_BINARY:
		if err := doBin(f); err != nil {
			errs = append(errs, err)
		}
	case FILE_RPM:
		if err := doRPM(f); err != nil {
			errs = append(errs, err)
		}
	case FILE_REGULAR:
	}
	return errs.HasError()
}

func doRPM(f *File) error {

	rpm := filepath.Join(f.extractDir(), FILE_RPM)
	info, err := ioutil.ReadDir(rpm)
	if err != nil {
		return err
	}
	var rpms []string
	rpms = append(rpms, "localinstall", "-y")
	for _, i := range info {
		if i.IsDir() {
			continue
		}
		rpms = append(rpms, filepath.Join(rpm, i.Name()))
	}

	extract := <-cmd.NewCmd("yum", rpms...).Start()
	return cmd.CmdError(extract)
}

func doBin(f *File) error {
	bin := filepath.Join(f.extractDir(), FILE_BINARY)
	dirs, err := ioutil.ReadDir(bin)
	if err != nil {
		return fmt.Errorf("list file error: %s, %s", bin, err.Error())
	}
	for _, v := range dirs {
		if v.IsDir() {
			continue
		}
		path := filepath.Join(bin, v.Name())
		status := <-cmd.NewCmd(
			"chmod",
			"+x", path,
		).Start()
		if err := cmd.CmdError(status); err != nil {
			return err
		}
		dest := f.VersionedPath.Destination
		if dest == "" {
			dest = "/usr/bin"
		}
		if err := os.MkdirAll(dest, 0755); err != nil {
			return fmt.Errorf("enusre dire %s : %s", dest, err.Error())
		}
		err := os.Rename(
			filepath.Join(path),
			filepath.Join(dest, v.Name()),
		)
		if err != nil {
			return fmt.Errorf("mv file error: %s", err.Error())
		}
	}

	return nil
}

func NewFile(
	base string,
	dest string,
) Transfer {
	return Transfer{
		Base:  base,
		Cache: dest,
	}
}
