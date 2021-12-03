package api

import (
	"github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"k8s.io/klog/v2"
	"net/http"
	"strings"

	"fmt"
	"github.com/aoxn/ovm/pkg/context"
)

func Credentials(
	cache *context.CachedContext,
	write http.ResponseWriter,
	request *http.Request) {

	id := Parameter(request, "id")
	if id == "" {
		HttpResponseJson(write, fmt.Errorf("empty node id"), http.StatusInternalServerError)
		return
	}
	var (
		rnode   *v1.Master
		address []string
	)
	cache.Nodes.Range(
		func(key, value interface{}) bool {
			k, ok := key.(string)
			if !ok {
				klog.Infof("not type string. %v", k)
				return false
			}
			if k == id {
				rnode = value.(*v1.Master)
			}
			v := value.(*v1.Master)
			if v.Spec.Role == v1.NODE_ROLE_HYBRID ||
				v.Spec.Role == v1.NODE_ROLE_ETCD {
				address = append(address, v.Spec.IP)
			}
			return true
		},
	)
	if rnode == nil {
		HttpResponseJson(write, "not found", http.StatusNotFound)
		return
	}
	if err := validate(rnode); err != nil {
		HttpResponseJson(write, err, http.StatusConflict)
		return
	}
	cache.BootCFG.Etcd.Endpoints = strings.Join(address, ",")

	rnode.Status.BootCFG = v1.NewDefaultCluster("kubernetes-cluster", *cache.BootCFG)
	HttpResponseJson(write, rnode, http.StatusOK)
}
