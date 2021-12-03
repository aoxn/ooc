package addons

var INGRESS = ConfigTpl{
	Name:         "ingress-controller",
	Tpl:          ingress,
	Replicas:     "2",
	ImageVersion: "v0.22.0.5-552e0db-aliyun",
}

var ingress = `
{{ if eq .Action "Install" }}
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-ingress-lb
  namespace: {{.Namespace}}
  labels:
    app: nginx-ingress-lb
  {{ if .IngressSlbNetworkType }}
  annotations:
    service.beta.kubernetes.io/alicloud-loadbalancer-address-type: "{{.IngressSlbNetworkType}}"
  {{ end }}
spec:
  type: LoadBalancer
  externalTrafficPolicy: "Local"
  ports:
  - port: 80
    name: http
    targetPort: 80
  - port: 443
    name: https
    targetPort: 443
  selector:
    app: ingress-nginx
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-configuration
  namespace: {{.Namespace}}
  labels:
    app: ingress-nginx
data:
    log-format-upstream: '$the_real_ip - [$the_real_ip] - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $request_length $request_time [$proxy_upstream_name] $upstream_addr $upstream_response_length $upstream_response_time $upstream_status $req_id $host'
    proxy-body-size: 20m
    proxy-connect-timeout: "10"
    max-worker-connections: "65536"
    enable-underscores-in-headers: "true"
    reuse-port: "true"
    worker-cpu-affinity: "auto"
    server-tokens: "false"
    ssl-redirect: "false"
    allow-backend-server-header: "true"
    ignore-invalid-headers: "true"
    generate-request-id: "true"
    #forwarded-for-header: "X-Real-IP"
    #compute-full-forwarded-for: "true"
    #hsts: "false"
    #enable-vts-status: "true"
    #use-proxy-protocol: "true"
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: tcp-services
  namespace: {{.Namespace}}
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: udp-services
  namespace: {{.Namespace}}
{{ end }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nginx-ingress-controller
  namespace: {{.Namespace}}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: nginx-ingress-controller
rules:
  - apiGroups:
      - ""
    resources:
      - configmaps
      - endpoints
      - nodes
      - pods
      - secrets
      - namespaces
      - services
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - "extensions"
    resources:
      - ingresses
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - events
    verbs:
      - create
      - patch
  - apiGroups:
      - "extensions"
    resources:
      - ingresses/status
    verbs:
      - update
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - create
  - apiGroups:
      - ""
    resources:
      - configmaps
    resourceNames:
      - "ingress-controller-leader-nginx"
    verbs:
      - get
      - update
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: nginx-ingress-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nginx-ingress-controller
subjects:
  - kind: ServiceAccount
    name: nginx-ingress-controller
    namespace: {{.Namespace}}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-ingress-controller
  labels:
    app: ingress-nginx
  namespace: {{.Namespace}}
  annotations:
    component.version: "{{.ComponentRevision}}"
    component.revision: "{{.ComponentRevision}}"
spec:
  replicas: {{.Replicas}}
  selector:
    matchLabels:
      app: ingress-nginx
  template:
    metadata:
      labels:
        app: ingress-nginx
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
        prometheus.io/port: "10254"
        prometheus.io/scrape: "true"
    spec:
      #tolerations:
      #  - key: node-role.kubernetes.io/master
      #    effect: NoSchedule
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - ingress-nginx
              topologyKey: "kubernetes.io/hostname"
      serviceAccountName: nginx-ingress-controller
      initContainers:
      - name: init-sysctl
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/busybox:v1.29.2
        command:
        - /bin/sh
        - -c
        - |
          mount -o remount rw /proc/sys
          sysctl -w net.core.somaxconn=65535
          sysctl -w net.ipv4.ip_local_port_range="1024 65535"
          sysctl -w fs.file-max=1048576
          sysctl -w fs.inotify.max_user_instances=16384
          sysctl -w fs.inotify.max_user_watches=524288
          sysctl -w fs.inotify.max_queued_events=16384
        securityContext:
          capabilities:
              drop:
              - ALL
              add:
              - SYS_ADMIN
      containers:
      - name: nginx-ingress-controller
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/aliyun-ingress-controller:{{.ImageVersion}}
        args:
          - /nginx-ingress-controller
          - --configmap=$(POD_NAMESPACE)/nginx-configuration
          - --tcp-services-configmap=$(POD_NAMESPACE)/tcp-services
          - --udp-services-configmap=$(POD_NAMESPACE)/udp-services
          - --annotations-prefix=nginx.ingress.kubernetes.io
          - --publish-service=$(POD_NAMESPACE)/nginx-ingress-lb
          - --v=2
        env:
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: POD_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        ports:
        - name: http
          containerPort: 80
        - name: https
          containerPort: 443
        resources:
          requests:
            cpu: 100m
            memory: 70Mi
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 10254
            scheme: HTTP
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 10
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 10254
            scheme: HTTP
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 10
        securityContext:
          capabilities:
              drop:
              - ALL
              add:
              - NET_BIND_SERVICE
          runAsUser: 33
        volumeMounts:
        - name: localtime
          mountPath: /etc/localtime
          readOnly: true
      nodeSelector:
        beta.kubernetes.io/os: linux
      volumes:
        - name: localtime
          hostPath:
            path: /etc/localtime
            type: File
`
