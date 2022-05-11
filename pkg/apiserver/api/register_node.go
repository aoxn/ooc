package api

import (
	"fmt"
	v12 "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"time"
)

func RegisterNode(
	cache *context.CachedContext,
	write http.ResponseWriter,
	request *http.Request,
) {
	rnode, err := DecodeBody(request.Body)
	if err != nil {
		HttpResponseJson(write, err, http.StatusInternalServerError)
		return
	}
	if err := validate(rnode); err != nil {
		HttpResponseJson(write, err, http.StatusConflict)
		return
	}
	cache.SetKV(rnode.Spec.ID, rnode)
	HttpResponseJson(write, rnode, http.StatusOK)
}

func validate(node *v12.Master) error {
	if node.Name == "" {
		return fmt.Errorf("node.name can not be empty")
	}
	if node.Spec.ID == "" {
		return fmt.Errorf("node.spec.id can not be empty")
	}
	if node.Spec.IP == "" {
		return fmt.Errorf("node.spec.ip can not be empty")
	}
	if node.CreationTimestamp.IsZero() {
		node.CreationTimestamp = v1.NewTime(time.Now())
	}
	return nil
}
