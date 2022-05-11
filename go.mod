module github.com/aoxn/wdrip

go 1.16

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1204
	github.com/buger/goterm v0.0.0-20181115115552-c206103e1f37
	github.com/denverdino/aliyungo v0.0.0-20210729093130-35873bbdef21
	github.com/docker/distribution v2.7.1+incompatible
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a
	github.com/ghodss/yaml v1.0.0
	github.com/go-cmd/cmd v1.0.4
	github.com/go-test/deep v1.0.7 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/goterm v0.0.0-20200907032337-555d40f16ae2
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.3
	github.com/jonboulle/clockwork v0.1.0
	github.com/mdlayher/vsock v0.0.0-20210303205602-10d591861736
	github.com/moby/hyperkit v0.0.0-20211015224120-09fe9202a29a
	github.com/pkg/errors v0.9.1
	github.com/rs/xid v1.3.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/verybluebot/tarinator-go v0.0.0-20190613183509-5ab4e1193986
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	golang.org/x/sys v0.0.0-20211210111614-af8b64212486
	golang.org/x/tools v0.1.3 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/cli-runtime v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/cluster-bootstrap v0.0.0-00010101000000-000000000000
	k8s.io/code-generator v0.21.3
	k8s.io/klog/v2 v2.8.0
	k8s.io/kube-openapi v0.0.0-20210421082810-95288971da7e
	k8s.io/kubectl v0.21.0
	sigs.k8s.io/controller-runtime v0.9.5
	sigs.k8s.io/kustomize/kustomize/v4 v4.2.0
	sigs.k8s.io/kustomize/kyaml v0.11.0
)

replace (
	k8s.io/api => k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.3
	k8s.io/client-go => k8s.io/client-go v0.21.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.3
	k8s.io/code-generator => k8s.io/code-generator v0.21.3
	k8s.io/component-base => k8s.io/component-base v0.21.3
	k8s.io/kubectl => k8s.io/kubectl v0.21.3
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.8.11
	//sigs.k8s.io/kustomize/kustomize => sigs.k8s.io/kustomize/kustomize v4
)
