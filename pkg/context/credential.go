package context

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/context/base"
	"github.com/aoxn/ooc/pkg/utils/sign"
)

// CachedContext Nodes for nodes
type CachedContext struct {
	Nodes   *base.Context
	BootCFG *v1.ClusterSpec
}

func NewCachedContext(
	boot *v1.ClusterSpec,
) *CachedContext {
	return &CachedContext{
		Nodes:   base.NewContext(),
		BootCFG: boot,
	}
}

func NewKeyCert() (*v1.KeyCert, error) {
	key, crt, err := sign.SelfSignedPair()
	if err != nil {
		return nil, fmt.Errorf("self signed cert fail, %s", err.Error())
	}
	kv := &v1.KeyCert{
		Key:  key,
		Cert: crt,
	}
	return kv, nil
}

func NewKeyCertForSA() (*v1.KeyCert, error) {
	key, crt, err := sign.SelfSignedPairSA()
	if err != nil {
		return nil, fmt.Errorf("self signed sa cert fail, %s", err.Error())
	}
	kv := &v1.KeyCert{
		Key:  key,
		Cert: crt,
	}
	return kv, nil
}

func (n *CachedContext) GetMasters() []v1.Master {
	var masters []v1.Master
	n.Nodes.Range(
		func(key, value interface{}) bool {
			val,ok := value.(v1.Master)
			if ok {
				masters = append(masters, val)
			}
			return true
		},
	)
	return masters
}

func (n *CachedContext) AddMaster(m v1.Master) { n.Nodes.SetKV(m.Name, m) }

func (n *CachedContext) RemoveMaster(key string) { n.Nodes.Delete(key) }

func (n *CachedContext) Visit(set func(cache *CachedContext)) { set(n) }

func (n *CachedContext) SetKV(key string, value interface{}) { n.Nodes.SetKV(key, value) }
