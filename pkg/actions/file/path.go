package file

import (
	"fmt"
	"path/filepath"
)

// Path
// ${PROJECT}/${NAMESPACE}/${CLOUD_TYPE}/${PKG}/${VERSION}/${ARCH}/${OS}/files/
type Path struct {
	Project   string
	Namespace string
	// CloudType private public
	CType   string
	Pkg     string
	Version string
	Arch    string
	OS      string
	Ftype   string

	Destination string
}

// BinarySource return binary source path
// source: ${PROJECT}/${NAMESPACE}/${CLOUD_TYPE}/${PKG}/${VERSION}/${ARCH}/${OS}/files/
// source: wdripaoxn/public/kubernetes/1.12.6-aliyun.1/amd64/linux/files/{kubelet,kubectl,kubeadm}
func (p *Path) URL() string {
	return filepath.Join(
		p.Project, p.Namespace, p.CType, p.Pkg, p.Version, p.Arch, p.OS,
	)
}

func (p *Path) URL_T() string {
	return filepath.Join(p.URL(), p.Ftype)
}

func (p *Path) URI() string {
	return filepath.Join(p.URL(), p.Name())
}

func (p *Path) Name() string {
	return fmt.Sprintf("%s-%s.tar.gz", p.Pkg, p.Version)
}
