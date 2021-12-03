package auth

import (
	"k8s.io/klog/v2"
	"net/http"
)

type Authenticate interface {
	Authorize(req *http.Request) error
}

type TokenAuthenticator struct {
}

func (auth *TokenAuthenticator) Authorize(req *http.Request) error {
	klog.Infof("pass authentication")
	return nil
}
