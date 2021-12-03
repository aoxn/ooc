package api

import (
	"fmt"
	v1 "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/context"
	"net/http"
)

func Cluster(
	cache *context.CachedContext,
	write http.ResponseWriter,
	request *http.Request) {

	id := Parameter(request, "id")
	if id == "" {
		HttpResponseJson(write, fmt.Errorf("empty node id"), http.StatusInternalServerError)
		return
	}

	spec := v1.NewDefaultCluster("kubernetes-cluster", *cache.BootCFG)
	for _, m := range cache.GetMasters() {
		spec.Status.Peers = append(spec.Status.Peers, v1.Host{IP: m.Spec.IP, ID: m.Spec.ID})
	}
	HttpResponseJson(write, spec, http.StatusOK)
}
