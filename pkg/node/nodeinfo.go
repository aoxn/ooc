package node

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/node/meta"
)

func NewNodeInfo(
	meta meta.Meta,
) *NodeInfo {
	return &NodeInfo{meta: meta}
}

type Interface interface {
	NodeID() (string, error)
	NodeIP() (string, error)
	OS() (string, error)
	Arch() (string, error)
	Region() (string, error)
}

type NodeInfo struct {
	meta meta.Meta
}

func (i *NodeInfo) NodeID() (string, error) {
	id, err := i.meta.InstanceID()
	if err != nil {
		return "", err
	}
	region, err := i.meta.Region()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", region, id), nil
}

func (i *NodeInfo) NodeIP() (string, error) { return i.meta.PrivateIPv4() }

func (i *NodeInfo) Region() (string, error) { return i.meta.Region() }

func (i *NodeInfo) Arch() (string, error) { return "amd64", nil }

func (i *NodeInfo) OS() (string, error) { return "centos", nil }
