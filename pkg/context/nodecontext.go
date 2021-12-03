package context

import (
	"fmt"
	"github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/client/ovm"
	"github.com/aoxn/ovm/pkg/context/base"
	"github.com/aoxn/ovm/pkg/iaas/provider"
	"github.com/aoxn/ovm/pkg/node"
	"github.com/aoxn/ovm/pkg/node/meta/alibaba"
)

const (
	OvmFlags     = "OvmFlags"
	NodeMetaData = "NodeMetaData"
	ProviderCtx          = "ProviderCtx"
	ClusterClient        = "ClusterClient"
	BootNodeClient       = "BootNodeClient"
	NodeInfoObject       = "NodeInfoObject"
	BootCredentialClient = "BootCredentialClient"
)

func NewNodeContext(
	flags v1.OvmOptions,
) (*NodeContext, error) {
	ctxs := NodeContext{}
	if flags.Endpoint != "" {
		restc, err := ovm.RestClientForOVM(flags.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("construct rest client: %s", err.Error())
		}
		if restc != nil {
			ctxs.SetKV(BootNodeClient, ovm.NewNodeClient(restc))
			ctxs.SetKV(ClusterClient, ovm.NewClusterClient(restc))
			ctxs.SetKV(BootCredentialClient, ovm.NewCredential(restc))
		}
	}
	// meta provider factory
	meta := alibaba.NewMetaDataAlibaba(nil)

	ctxs.SetKV(OvmFlags, flags)
	ctxs.SetKV(NodeMetaData, node.NewNodeInfo(meta))
	return &ctxs, nil
}

type NodeContext struct{ base.Context }

func (c *NodeContext) NodeMetaData() node.Interface {
	return c.Value(NodeMetaData).(node.Interface)
}

func (c *NodeContext) ProviderCtx() *provider.Context {
	return c.Value(ProviderCtx).(*provider.Context)
}

// BootNodeClient ovm bootstrap client
func (c *NodeContext) BootNodeClient() ovm.Interface {
	return c.Value(BootNodeClient).(ovm.Interface)
}

// BootClusterClient ovm bootstrap client
func (c *NodeContext) BootClusterClient() ovm.InterfaceCluster {
	return c.Value(ClusterClient).(ovm.InterfaceCluster)
}

// BootCredentialClient ovm bootstrp client for credentialcfg
func (c *NodeContext) BootCredentialClient() ovm.InterfaceCredential {
	return c.Value(BootCredentialClient).(ovm.InterfaceCredential)
}

// NodeObject for NodeInfo object
func (c *NodeContext) NodeObject() *v1.Master {
	return c.Value(NodeInfoObject).(*v1.Master)
}

// ActionContext
func (c *NodeContext) OvmFlags() v1.OvmOptions {
	return c.Value(OvmFlags).(v1.OvmOptions)
}

// ActionContext
func (c *NodeContext) ExpectedMasterCnt() int {
	return c.Value(OvmFlags).(v1.OvmOptions).ExpectedMasterCnt
}
