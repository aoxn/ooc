package addons

var KUBEPROXY_MASTER = ConfigTpl{
	Name:         "kubeproxy-master",
	Tpl:          kubeproxyMaster,
	ImageVersion: "v1.16.9-aliyun.1",
}

var KUBEPROXY_WORKER = ConfigTpl{
	Name:         "kubeproxy-worker",
	Tpl:          kubeproxyWorker,
	ImageVersion: "v1.16.9-aliyun.1",
}

var kubeproxyMaster = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    k8s-app: kube-proxy-master
  name: kube-proxy-master
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: kube-proxy-master
  template:
    metadata:
      labels:
        k8s-app: kube-proxy-master
    spec:
      priorityClassName: system-node-critical
      containers:
      - command:
        - /usr/local/bin/kube-proxy
{{if .ProxyMode }}
        - --proxy-mode={{.ProxyMode}}
{{end}}
        - --kubeconfig=/var/lib/kube-proxy/kubeconfig.conf
        - --cluster-cidr={{.CIDR}}
        - --hostname-override=$(NODE_NAME)
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/kube-proxy:{{.ImageVersion}}
        imagePullPolicy: IfNotPresent
        name: kube-proxy-master
        resources: {}
        securityContext:
          privileged: true
        terminationMessagePath: /alibaba/termination-log
        terminationMessagePolicy: File
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - mountPath: /var/lib/kube-proxy
          name: kube-proxy-master
        - mountPath: /run/xtables.lock
          name: xtables-lock
        - mountPath: /lib/modules
          name: lib-modules
          readOnly: true
      dnsPolicy: ClusterFirst
      hostNetwork: true
      restartPolicy: Always
      serviceAccountName: kube-proxy
      terminationGracePeriodSeconds: 30
      tolerations:
      - operator: Exists
      nodeSelector:
        node-role.kubernetes.io/master: ""
      volumes:
      - configMap:
          defaultMode: 420
          name: kube-proxy-master
        name: kube-proxy-master
      - hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
        name: xtables-lock
      - hostPath:
          path: /lib/modules
          type: ""
        name: lib-modules
  updateStrategy:
    type: RollingUpdate

{{ if eq .Action "Install" }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-proxy
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubeadm:node-proxier
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:node-proxier
subjects:
- kind: ServiceAccount
  name: kube-proxy
  namespace: kube-system
---
## deploy kube-proxy on master with apiserver connect to 127.0.0.1:6443
apiVersion: v1
data:
  kubeconfig.conf: |
    apiVersion: v1
    kind: Config
    clusters:
    - cluster:
        server: https://127.0.0.1:6443
        insecure-skip-tls-verify: true
      name: default
    contexts:
    - context:
        cluster: default
        namespace: default
        user: default
      name: default
    current-context: default
    users:
    - name: default
      user:
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
kind: ConfigMap
metadata:
  labels:
    app: kube-proxy-master
  name: kube-proxy-master
  namespace: kube-system
{{ end }}
`

var kubeproxyWorker = `
## Avoid SLB loopback. make this change .
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    k8s-app: kube-proxy-worker
  name: kube-proxy-worker
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: kube-proxy-worker
  template:
    metadata:
      labels:
        k8s-app: kube-proxy-worker
    spec:
      priorityClassName: system-node-critical
      containers:
      - command:
        - /usr/local/bin/kube-proxy
{{if .ProxyMode }}
        - --proxy-mode={{.ProxyMode}}
{{end}}
        - --kubeconfig=/var/lib/kube-proxy/kubeconfig.conf
        - --cluster-cidr={{.CIDR}}
        - --hostname-override=$(NODE_NAME)
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/kube-proxy:{{.ImageVersion}}
        imagePullPolicy: IfNotPresent
        name: kube-proxy-worker
        resources: {}
        securityContext:
          privileged: true
        terminationMessagePath: /alibaba/termination-log
        terminationMessagePolicy: File
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - mountPath: /var/lib/kube-proxy
          name: kube-proxy-worker
        - mountPath: /run/xtables.lock
          name: xtables-lock
        - mountPath: /lib/modules
          name: lib-modules
          readOnly: true
      dnsPolicy: ClusterFirst
      hostNetwork: true
      restartPolicy: Always
      serviceAccountName: kube-proxy
      terminationGracePeriodSeconds: 30
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: DoesNotExist
      tolerations:
      - operator: Exists
      volumes:
      - configMap:
          defaultMode: 420
          name: kube-proxy-worker
        name: kube-proxy-worker
      - hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
        name: xtables-lock
      - hostPath:
          path: /lib/modules
          type: ""
        name: lib-modules
  updateStrategy:
    type: RollingUpdate
{{ if eq .Action "Install" }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-proxy
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubeadm:node-proxier
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:node-proxier
subjects:
- kind: ServiceAccount
  name: kube-proxy
  namespace: kube-system

  ## deploy kube-proxy on worker with apiserver point to apiserver_lb.
---
apiVersion: v1
data:
  kubeconfig.conf: |
    apiVersion: v1
    kind: Config
    clusters:
    - cluster:
        certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        server: {{.IntranetApiServerEndpoint}}
      name: default
    contexts:
    - context:
        cluster: default
        namespace: default
        user: default
      name: default
    current-context: default
    users:
    - name: default
      user:
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
kind: ConfigMap
metadata:
  labels:
    app: kube-proxy-worker
  name: kube-proxy-worker
  namespace: kube-system
{{ end }}
`
