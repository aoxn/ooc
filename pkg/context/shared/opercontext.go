package shared

import (
	"github.com/aoxn/wdrip/pkg/context"
	"github.com/aoxn/wdrip/pkg/context/base"
	"github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/aoxn/wdrip/pkg/operator/heal"
)

const (
	NodeCacheCtx = "NodeCacheCtx"
	ProviderIAAS = "ProviderIAAS"
	MemberHeal   = "Healet"
	ProviderCtx  = "ProviderCtx"
)

func NewOperatorContext(
	cache *context.CachedContext,
	prvd provider.Interface,
	mem *heal.Healet,
	pctx *provider.Context,
) *SharedOperatorContext {
	ctxs := SharedOperatorContext{}
	ctxs.SetKV(ProviderIAAS, prvd)
	ctxs.SetKV(NodeCacheCtx, cache)
	ctxs.SetKV(MemberHeal, mem)
	ctxs.SetKV(ProviderCtx, pctx)
	return &ctxs
}

type SharedOperatorContext struct{ base.Context }

// metadata for cloud node
func (c *SharedOperatorContext) NodeCacheCtx() *context.CachedContext {
	return c.Value(NodeCacheCtx).(*context.CachedContext)
}

func (c *SharedOperatorContext) ProvdIAAS() provider.Interface {
	return c.Value(ProviderIAAS).(provider.Interface)
}

func (c *SharedOperatorContext) MemberHeal() *heal.Healet {
	return c.Value(MemberHeal).(*heal.Healet)
}

func (c *SharedOperatorContext) ProviderCtx() *provider.Context {
	return c.Value(ProviderCtx).(*provider.Context)
}
