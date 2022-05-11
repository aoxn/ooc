package wdrip

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/client/rest"
)

type Interface interface {
	Create(node *v1.Master) (*v1.Master, error)
	Update(*v1.Master) (*v1.Master, error)
	Delete(name string) error
	Get(name string) (*v1.Master, error)
	List() (*v1.MasterList, error)
}

type Node struct {
	client rest.Interface
}

func NewNodeClient(
	client rest.Interface,
) *Node {
	return &Node{
		client: client,
	}
}

func (m *Node) Get(name string) (*v1.Master, error) {
	node := v1.Master{}
	err := m.client.
		Get().
		PathPrefix("/api/v1").
		Resource("nodes").
		ResourceName(name).
		Do(&node)
	return &node, err
}

func (m *Node) Create(node *v1.Master) (*v1.Master, error) {
	rnode := v1.Master{}
	err := m.client.
		Post().
		PathPrefix("/api/v1").
		Resource("nodes").
		Body(node).
		Do(&rnode)
	return &rnode, err
}

func (m *Node) List() (*v1.MasterList, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (m *Node) Delete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (m *Node) Update(*v1.Master) (*v1.Master, error) {
	return nil, fmt.Errorf("unimplemented")
}
