package wdrip

import (
	"github.com/aoxn/wdrip/pkg/client/rest"
)

func RestClientForWDRIP(endpoint string) (rest.Interface, error) {
	return rest.RESTClientFor(
		&rest.Config{
			Host:        endpoint,
			ContentType: "application/json",
			UserAgent:   "kubernetes.wdrip",
		},
	)
}
