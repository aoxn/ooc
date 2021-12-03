package boot

import (
	"fmt"
	v1 "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/context"
	"github.com/docker/distribution/uuid"
	"k8s.io/cluster-bootstrap/token/util"
)

func SetDefaultCredential(spec *v1.ClusterSpec) {
	if spec.Kubernetes.RootCA == nil {
		root, err := context.NewKeyCert()
		if err != nil {
			panic("new self signed cert pair fail")
		}
		spec.Kubernetes.RootCA = root
	}
	if spec.Kubernetes.FrontProxyCA == nil {
		front, err := context.NewKeyCert()
		if err != nil {
			panic("new self signed cert pair fail")
		}
		spec.Kubernetes.FrontProxyCA = front
	}
	if spec.Kubernetes.SvcAccountCA == nil {
		sa, err := context.NewKeyCertForSA()
		if err != nil {
			panic("new self signed cert pair fail")
		}
		spec.Kubernetes.SvcAccountCA = sa
	}
	if spec.Kubernetes.KubeadmToken == "" {
		token, err := util.GenerateBootstrapToken()
		if err != nil {
			panic(fmt.Sprintf("token generate: %s", err.Error()))
		}
		spec.Kubernetes.KubeadmToken = token
	}
	if spec.Etcd.ServerCA == nil {
		serca, err := context.NewKeyCert()
		if err != nil {
			panic("new self signed cert pair fail")
		}
		spec.Etcd.ServerCA = serca
	}
	if spec.Etcd.PeerCA == nil {
		serca, err := context.NewKeyCert()
		if err != nil {
			panic("new self signed cert pair fail")
		}
		spec.Etcd.PeerCA = serca
	}
	if spec.Etcd.InitToken == "" {
		spec.Etcd.InitToken = uuid.Generate().String()
	}
}
