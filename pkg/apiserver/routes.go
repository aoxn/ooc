package apiserver

import (
	"k8s.io/klog/v2"
	"net/http"

	"github.com/aoxn/wdrip/pkg/apiserver/api"
	"github.com/gorilla/mux"
)

var routes = map[string]map[string]api.HandlerFunc{
	"GET": {
		"/debug":                   api.Debug,
		"/api/v1/nodes":            api.Debug,
		"/fake":                    api.FakeHandler,
		"/api/v1/credentials/{id}": api.Credentials,
		"/api/v1/clusters/{id}":    api.Cluster,
	},
	"PUT": {},
	"POST": {
		"/api/v1/nodes": api.RegisterNode,
	},
	"DELETE": {
		"/api/v1/nodes/{id}": api.FakeHandler,
	},
}

func SetupRoutes(server *Server, muxs *mux.Router) {
	for method, mappings := range routes {
		for r, h := range mappings {
			handler := h

			klog.Infof("start to register http router: %s", r)

			muxs.Path(r).Methods(method).HandlerFunc(
				func(w http.ResponseWriter, req *http.Request) {

					klog.Infof("receive request")

					if err := server.Auth.Authorize(req); err != nil {
						http.Error(w, "authentication failure", http.StatusBadRequest)
						return
					}
					handler(server.CachedCtx, w, req)
				},
			)
		}
	}
	server.Handler = muxs
}
