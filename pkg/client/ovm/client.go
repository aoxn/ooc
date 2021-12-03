package ovm

import (
	"github.com/aoxn/ovm/pkg/client/rest"
)

func RestClientForOVM(endpoint string) (rest.Interface, error) {
	return rest.RESTClientFor(
		&rest.Config{
			Host:        endpoint,
			ContentType: "application/json",
			UserAgent:   "kubernetes.ovm",
		},
	)
}
