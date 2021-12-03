package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/utils"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"

	"github.com/aoxn/ovm/pkg/context"
)

type HandlerFunc func(contex *context.CachedContext, w http.ResponseWriter, r *http.Request)

func FakeHandler(
	contex *context.CachedContext,
	w http.ResponseWriter,
	req *http.Request) {

	HttpResponseJson(w, Status{
		HttpCode: http.StatusOK,
		Message:  "FakeHandler",
		Status:   "Success",
	}, http.StatusOK)
}

func Debug(
	cache *context.CachedContext,
	w http.ResponseWriter,
	req *http.Request) {
	var nodes []*v1.Master
	cache.Nodes.Range(
		func(key, value interface{}) bool {
			k := key.(string)
			klog.Infof("==============================================================")
			klog.Infof(k)
			klog.Infof("%s\n\n", utils.PrettyYaml(value))
			v := value.(v1.Master)
			nodes = append(nodes, &v)
			return true
		},
	)

	HttpResponseJson(w, nodes, http.StatusOK)
}

func Parameter(r *http.Request, key string) string { return mux.Vars(r)[key] }

func DecodeBody(body io.ReadCloser) (*v1.Master, error) {
	node := &v1.Master{}
	result, err := ioutil.ReadAll(body)
	if err != nil {
		return node, err
	}
	if err := json.Unmarshal(result, node); err != nil {

		return nil, fmt.Errorf("unmarshal node spec error: %s %s", err.Error(), result)
	}
	klog.Infof("=====================================================================\n")
	klog.Infof(fmt.Sprintf("Body:\n%s\n\n", utils.PrettyYaml(node)))
	return node, nil
}

type Status struct {
	HttpCode int    `json:"code,omitempty" protobuf:"bytes,1,opt,name=code"`
	Status   string `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
	Message  string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
}

func responseText(v interface{}) string {
	msg, _ := json.Marshal(v)
	return string(msg)
}

func HttpResponseJson(w http.ResponseWriter, v interface{}, code int) int {
	msg := "ok"
	if err, ok := v.(error); ok {
		msg = responseText(
			Status{
				HttpCode: code,
				Status:   "Error",
				Message:  err.Error(),
			},
		)
	} else {
		msg = responseText(v)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	io.Copy(w, bytes.NewBuffer([]byte(msg)))
	return code
}
