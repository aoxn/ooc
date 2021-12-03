package context

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/client/ooc"
	"github.com/aoxn/ooc/pkg/context/base"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/node"
	"github.com/aoxn/ooc/pkg/node/meta/alibaba"
)

const (
	OocFlags             = "OocFlags"
	NodeMetaData         = "NodeMetaData"
	ProviderCtx          = "ProviderCtx"
	ClusterClient        = "ClusterClient"
	BootNodeClient       = "BootNodeClient"
	NodeInfoObject       = "NodeInfoObject"
	BootCredentialClient = "BootCredentialClient"
)

func NewNodeContext(
	flags v1.OocOptions,
) (*NodeContext, error) {
	ctxs := NodeContext{}
	if flags.Endpoint != "" {
		restc, err := ooc.RestClientForOOC(flags.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("construct rest client: %s", err.Error())
		}
		if restc != nil {
			ctxs.SetKV(BootNodeClient, ooc.NewNodeClient(restc))
			ctxs.SetKV(ClusterClient, ooc.NewClusterClient(restc))
			ctxs.SetKV(BootCredentialClient, ooc.NewCredential(restc))
		}
	}
	// meta provider factory
	meta := alibaba.NewMetaDataAlibaba(nil)

	ctxs.SetKV(OocFlags, flags)
	ctxs.SetKV(NodeMetaData, node.NewNodeInfo(meta))
	return &ctxs, nil
}

type NodeContext struct{ base.Context }

// metadata for cloud node
func (c *NodeContext) NodeMetaData() node.Interface {
	return c.Value(NodeMetaData).(node.Interface)
}

func (c *NodeContext) ProviderCtx() provider.Interface {
	return c.Value(ProviderCtx).(provider.Interface)
}

// BootNodeClient ooc bootstrap client
func (c *NodeContext) BootNodeClient() ooc.Interface {
	return c.Value(BootNodeClient).(ooc.Interface)
}

// BootClusterClient ooc bootstrap client
func (c *NodeContext) BootClusterClient() ooc.InterfaceCluster {
	return c.Value(ClusterClient).(ooc.InterfaceCluster)
}

// BootCredentialClient ooc bootstrp client for credentialcfg
func (c *NodeContext) BootCredentialClient() ooc.InterfaceCredential {
	return c.Value(BootCredentialClient).(ooc.InterfaceCredential)
}

// NodeObject for NodeInfo object
func (c *NodeContext) NodeObject() *v1.Master {
	return c.Value(NodeInfoObject).(*v1.Master)
}

// ActionContext
func (c *NodeContext) OocFlags() v1.OocOptions {
	return c.Value(OocFlags).(v1.OocOptions)
}

// ActionContext
func (c *NodeContext) ExpectedMasterCnt() int {
	return c.Value(OocFlags).(v1.OocOptions).ExpectedMasterCnt
}
