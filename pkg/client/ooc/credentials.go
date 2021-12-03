package ooc

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/client/rest"
)

type InterfaceCredential interface {
	Create(node *v1.Master) (*v1.Master, error)
	Update(*v1.Master) (*v1.Master, error)
	Delete(name string) error
	Get(name string) (*v1.Master, error)
	List() (*v1.MasterList, error)
}

type Credential struct {
	client rest.Interface
}

func NewCredential(
	client rest.Interface,
) *Credential {
	return &Credential{
		client: client,
	}
}

func (m *Credential) Get(name string) (*v1.Master, error) {
	node := v1.Master{}
	err := m.client.
		Get().
		PathPrefix("/api/v1").
		Resource("credentials").
		ResourceName(name).
		Do(&node)
	return &node, err
}

func (m *Credential) Create(node *v1.Master) (*v1.Master, error) {
	rnode := v1.Master{}
	err := m.client.
		Post().
		PathPrefix("/api/v1").
		Resource("credentials").
		Body(node).
		Do(&node)
	return &rnode, err
}

func (m *Credential) Delete(name string) error {
	return fmt.Errorf("unimplemented")
}

func (m *Credential) Update(*v1.Master) (*v1.Master, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (m *Credential) List() (*v1.MasterList, error) {
	return nil, fmt.Errorf("unimplemented")
}
