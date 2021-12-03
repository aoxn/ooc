package ooc

import (
	"github.com/aoxn/ooc/pkg/client/rest"
)

func RestClientForOOC(endpoint string) (rest.Interface, error) {
	return rest.RESTClientFor(
		&rest.Config{
			Host:        endpoint,
			ContentType: "application/json",
			UserAgent:   "kubernetes.ooc",
		},
	)
}
