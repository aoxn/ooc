package post

import (
	"bytes"
	"fmt"
	"github.com/aoxn/wdrip"
	v12 "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/wdrip/pkg/utils"
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
	t, err := template.New("wdrip-file").Parse(monitor)
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
			Version:     wdrip.Version,
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
    app: wdrip-monitor
  name: wdrip-monitor
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: wdrip-monitor
  template:
    metadata:
      labels:
        app: wdrip-monitor
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
        - image: {{ .Registry }}/wdrip:{{ .Version }}
          imagePullPolicy: Always
          name: wdrip-monitor-net
          command:
            - /wdrip
            - monit
            - -n="{{.ClusterName}}"
          volumeMounts:
            - name: bootcfg
              mountPath: /etc/wdrip/
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
