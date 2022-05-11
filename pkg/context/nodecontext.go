package context

import (
	"fmt"
	"github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/client/wdrip"
	"github.com/aoxn/wdrip/pkg/context/base"
	"github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/aoxn/wdrip/pkg/node"
	"github.com/aoxn/wdrip/pkg/node/meta/alibaba"
)

const (
	WdripFlags           = "WdripFlags"
	NodeMetaData         = "NodeMetaData"
	ProviderCtx          = "ProviderCtx"
	ClusterClient        = "ClusterClient"
	BootNodeClient       = "BootNodeClient"
	NodeInfoObject       = "NodeInfoObject"
	BootCredentialClient = "BootCredentialClient"
)

func NewNodeContext(
	flags v1.WdripOptions,
) (*NodeContext, error) {
	ctxs := NodeContext{}
	if flags.Endpoint != "" {
		restc, err := wdrip.RestClientForWDRIP(flags.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("construct rest client: %s", err.Error())
		}
		if restc != nil {
			ctxs.SetKV(BootNodeClient, wdrip.NewNodeClient(restc))
			ctxs.SetKV(ClusterClient, wdrip.NewClusterClient(restc))
			ctxs.SetKV(BootCredentialClient, wdrip.NewCredential(restc))
		}
	}
	// meta provider factory
	meta := alibaba.NewMetaDataAlibaba(nil)

	ctxs.SetKV(WdripFlags, flags)
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

// BootNodeClient wdrip bootstrap client
func (c *NodeContext) BootNodeClient() wdrip.Interface {
	return c.Value(BootNodeClient).(wdrip.Interface)
}

// BootClusterClient wdrip bootstrap client
func (c *NodeContext) BootClusterClient() wdrip.InterfaceCluster {
	return c.Value(ClusterClient).(wdrip.InterfaceCluster)
}

// BootCredentialClient wdrip bootstrp client for credentialcfg
func (c *NodeContext) BootCredentialClient() wdrip.InterfaceCredential {
	return c.Value(BootCredentialClient).(wdrip.InterfaceCredential)
}

// NodeObject for NodeInfo object
func (c *NodeContext) NodeObject() *v1.Master {
	return c.Value(NodeInfoObject).(*v1.Master)
}

// ActionContext
func (c *NodeContext) WdripFlags() v1.WdripOptions {
	return c.Value(WdripFlags).(v1.WdripOptions)
}

// ActionContext
func (c *NodeContext) ExpectedMasterCnt() int {
	return c.Value(WdripFlags).(v1.WdripOptions).ExpectedMasterCnt
}
