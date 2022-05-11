package wdrip

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/client/rest"
)

type InterfaceCluster interface {
	Create(node *v1.Cluster) (*v1.Cluster, error)
	Update(*v1.Cluster) (*v1.Cluster, error)
	Delete(name string) error
	Get(name string) (*v1.Cluster, error)
	List() (*v1.ClusterList, error)
}

type Cluster struct {
	client rest.Interface
}

func NewClusterClient(
	client rest.Interface,
) *Cluster {
	return &Cluster{
		client: client,
	}
}

func (m *Cluster) Get(name string) (*v1.Cluster, error) {
	node := v1.Cluster{}
	err := m.client.
		Get().
		PathPrefix("/api/v1").
		Resource("clusters").
		ResourceName(name).
		Do(&node)
	return &node, err
}

func (m *Cluster) Create(node *v1.Cluster) (*v1.Cluster, error) {
	rnode := v1.Cluster{}
	err := m.client.
		Post().
		PathPrefix("/api/v1").
		Resource("clusters").
		Body(node).
		Do(&node)
	return &rnode, err
}

func (m *Cluster) Delete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (m *Cluster) Update(cluster *v1.Cluster) (*v1.Cluster, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (m *Cluster) List() (*v1.ClusterList, error) {
	return nil, fmt.Errorf("unimplemented")
}
