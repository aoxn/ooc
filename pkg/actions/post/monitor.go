package post

import (
	"bytes"
	"fmt"
	"github.com/aoxn/ovm"
	v12 "github.com/aoxn/ovm/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ovm/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"path/filepath"
	"text/template"
	"time"
)

func RunMonitor(ctx *v12.ClusterSpec) error {
	cfg, err := RenderMonitorYaml(ctx)
	if err != nil {
		return fmt.Errorf("write monitor yaml: %s", err.Error())
	}
	return wait.Poll(
		2*time.Second,
		1*time.Minute,
		func() (done bool, err error) {
			if err := BootCFG(ctx); err != nil {
				klog.Errorf("retry upload bootcfg fail: %s", err.Error())
				return false, nil
			}
			if err := utils.ApplyYaml(cfg, "monitor"); err != nil {
				klog.Errorf("retry wait for monitor addon: %s", err.Error())
				return false, nil
			}
			return true, nil
		},
	)
}

func RenderMonitorYaml(spec *v12.ClusterSpec) (string, error) {
	t, err := template.New("ovm-file").Parse(monitor)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(
		&buff,
		struct {
			Version     string
			Registry    string
			ClusterName string
		}{
			Version:     ovm.Version,
			ClusterName: spec.ClusterID,
			Registry:    fmt.Sprintf("%s/aoxn", filepath.Dir(spec.Registry)),
			//Registry: "registry.cn-hangzhou.aliyuncs.com/aoxn",
		},
	)
	return buff.String(), err
}

var (
	monitor = `
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: ovm-monitor
  name: ovm-monitor
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ovm-monitor
  template:
    metadata:
      labels:
        app: ovm-monitor
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: "node-role.kubernetes.io/master"
                operator: DoesNotExist
      hostNetwork: true
      serviceAccount: admin
      priorityClassName: system-node-critical
      containers:
        - image: {{ .Registry }}/ovm:{{ .Version }}
          imagePullPolicy: Always
          name: ovm-monitor-net
          command:
            - /ovm
            - monit
            - -n="{{.ClusterName}}"
          volumeMounts:
            - name: bootcfg
              mountPath: /etc/ovm/
              readOnly: true
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
