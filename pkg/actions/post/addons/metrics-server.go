package addons

var METRICS_SERVER = ConfigTpl{
	Name: 		  "metrics-server",
	Tpl:          metrics,
	ImageVersion: "v1.0.0.2-cc3b2d6-aliyun",
}
var metrics = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ags-metrics-collector
  labels:
    owner: aliyun
    app: ags-metrics-collector
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      owner: aliyun
      app: ags-metrics-collector
  template:
    metadata:
      labels:
        owner: aliyun
        app: ags-metrics-collector
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: k8s.io/cluster-autoscaler
                    operator: DoesNotExist
      containers:
        - name: ags-metrics-collector
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/ags-metrics-collector:{{.ImageVersion}}
          imagePullPolicy: Always
      serviceAccount: admin
      serviceAccountName: admin
`
