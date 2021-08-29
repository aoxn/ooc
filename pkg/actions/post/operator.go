//go:build linux || darwin
// +build linux darwin

package post

import (
	"bytes"
	"fmt"
	"github.com/aoxn/ooc"
	"github.com/aoxn/ooc/pkg/actions"
	"github.com/aoxn/ooc/pkg/actions/post/addons"
	v12 "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/context"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/aoxn/ooc/pkg/utils/crd"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"html/template"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"path/filepath"
	"strings"
	"time"
)

const (
	ObjectName        = "config"
	KUBELET_UNIT_FILE = "/etc/systemd/system/kubelet.service"
)

type ActionPost struct {
}

// NewAction returns a new ActionPost for post kubernetes install
func NewActionPost() actions.Action {
	return &ActionPost{}
}

// Execute runs the ActionPost
func (a *ActionPost) Execute(ctx *actions.ActionContext) error {
	// Addon was installed by operator
	adds := ctx.OocFlags().Addons
	cfgadds := []addons.ConfigTpl{addons.KUBEPROXY_MASTER}
	if adds == "*" {
		cfgadds = addons.AddonConfigsTpl()
	}
	err := addons.InstallAddons(ctx.Config(), cfgadds)
	if err != nil {
		return fmt.Errorf("install addons: %s", err.Error())
	}

	err = crd.RegisterFromKubeconfig("/etc/kubernetes/admin.conf")
	if err != nil {
		return fmt.Errorf("register crds: %s", err.Error())
	}
	err = WriteClusterInfo(ctx.NodeContext)
	if err != nil {
		return fmt.Errorf("write cluster cfg: %s", err.Error())
	}
	// Run ooc operator default
	return RunOoc(ctx.Config())
}

func WriteClusterInfo(ctx *context.NodeContext) error {

	cfg := ctx.NodeObject().Status.BootCFG
	m := ctx.NodeObject()
	node := v12.Master{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Master",
			APIVersion: v12.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: m.Spec.ID,
		},
		Spec: m.Spec,
	}
	return utils.ApplyYaml(
		strings.Join(
			[]string{
				utils.PrettyYaml(cfg),
				utils.PrettyYaml(node),
			}, "---\n",
		), "cluster-crd",
	)
}

func RunOoc(ctx *v12.ClusterSpec) error {
	cfg, err := RenderOocYaml(ctx)
	if err != nil {
		return fmt.Errorf("write ooc yaml: %s", err.Error())
	}
	return wait.Poll(
		2*time.Second,
		1*time.Minute,
		func() (done bool, err error) {
			if err := BootCFG(ctx); err != nil {
				klog.Errorf("retry upload bootcfg fail: %s", err.Error())
				return false, nil
			}
			if err := utils.ApplyYaml(cfg, "ooc"); err != nil {
				klog.Errorf("retry wait for ooc addon: %s", err.Error())
				return false, nil
			}
			return true, nil
		},
	)
}

func BootCFG(spec *v12.ClusterSpec) error {
	bootcfg, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal bootcfg: %s", err.Error())
	}
	cm := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bootcfg",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"bootcfg": bootcfg,
		},
	}

	cmdata, err := yaml.Marshal(cm)
	if err != nil {
		return fmt.Errorf("marshal cm: %s", err.Error())
	}
	return utils.ApplyYaml(string(cmdata), "bootcfg")
}

func RenderOocYaml(spec *v12.ClusterSpec) (string, error) {
	t, err := template.New("ooc-file").Parse(oocf)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(
		&buff,
		struct {
			Version  string
			Registry string
		}{
			Version:  ooc.Version,
			Registry: fmt.Sprintf("%s/aoxn", filepath.Dir(spec.Registry)),
			//Registry: "registry.cn-hangzhou.aliyuncs.com/aoxn",
		},
	)
	return buff.String(), err
}

var (
	oocf = `
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: admin
  namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: ooc
  name: ooc
  namespace: kube-system
spec:
  ports:
    - name: tcp
      nodePort: 32443
      port: 9443
      protocol: TCP
      targetPort: 443
  selector:
    app: ooc
  sessionAffinity: None
  type: NodePort
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: ooc
  name: ooc
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ooc
  template:
    metadata:
      labels:
        app: ooc
    spec:
      hostNetwork: true
      priorityClassName: system-node-critical
      serviceAccount: admin
      priorityClassName: system-node-critical
      containers:
        - image: {{ .Registry }}/ooc:{{ .Version }}
          imagePullPolicy: Always
          name: ooc-net
          command:
            - /ooc
            - operator
            # - --bootcfg=/etc/ooc/boot.cfg
          volumeMounts:
            - name: bootcfg
              mountPath: /etc/ooc/
              readOnly: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
        - operator: Exists
      volumes:
        - name: bootcfg
          secret:
            secretName: bootcfg
            items:
              - key: bootcfg
                path: boot.cfg
`
)
