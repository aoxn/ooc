package apiserver

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/apiserver/auth"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
	"net"
	"net/http"
)

// ServerConfig server config
type Configuration struct {
	BindAddr    string
	TLSKeyPath  string
	TLSCertPath string
	TLSCAPath   string
}

type Server struct {
	Handler   http.Handler
	CachedCtx *context.CachedContext
	Auth      auth.Authenticate
	Config    Configuration
}

func (v *Server) Start() error {
	if err := v.initialize(); err != nil {
		return err
	}

	listener, err := newListener(v.Config)
	if err != nil {
		return fmt.Errorf("create listener error with: %s", err.Error())
	}
	go func() {
		err := http.Serve(listener, v.Handler)
		if err != nil {
			klog.Errorf("run server: %s", err.Error())
		}
		klog.Info("server started...")
	}()
	return nil
}

func (v *Server) initialize() error {
	SetupRoutes(v, mux.NewRouter())
	return nil
}

func newListener(cfg Configuration) (net.Listener, error) {
	if cfg.TLSCAPath == "" ||
		cfg.TLSCertPath == "" ||
		cfg.TLSKeyPath == "" {
		return net.Listen("tcp", cfg.BindAddr)
	}
	return nil, nil
}
