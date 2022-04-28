package addons

var CSI_PLUGIN = ConfigTpl{
  Name:         "csi-plugin",
  Tpl:          csi_plugin,
  ImageVersion: "v1.20.4.0-65b2faa-aliyun",
}

var CSI_PROVISION = ConfigTpl{
  Name:         "csi-provision",
  Tpl:          csi_provision,
  ImageVersion: "v1.20.4.0-65b2faa-aliyun",
}

var csi_plugin = `
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-admin
  namespace: kube-system
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: alicloud-csi-plugin
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "update", "create", "delete", "patch"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims/status"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["csinodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "watch", "list", "delete", "update", "create"]
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["csi.storage.k8s.io"]
    resources: ["csinodeinfos"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents"]
    verbs: ["create", "get", "list", "watch", "update", "delete"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["create", "list", "watch", "delete", "get", "update", "patch"]
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "create", "list", "watch", "delete", "update"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshotcontents/status"]
    verbs: ["update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["volumeattachments/status"]
    verbs: ["patch"]
  - apiGroups: ["snapshot.storage.k8s.io"]
    resources: ["volumesnapshots/status"]
    verbs: ["update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods","pods/exec"]
    verbs: ["create", "delete", "get", "post", "list", "watch", "patch", "udpate"]
  - apiGroups: ["storage.alibabacloud.com"]
    resources: ["rules"]
    verbs: ["get"]
  - apiGroups: ["storage.alibabacloud.com"]
    resources: ["containernetworkfilesystems"]
    verbs: ["get","list", "watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: alicloud-csi-plugin
subjects:
  - kind: ServiceAccount
    name: csi-admin
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: alicloud-csi-plugin
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: storage.k8s.io/v1beta1
kind: CSIDriver
metadata:
  name: diskplugin.csi.alibabacloud.com
spec:
  attachRequired: true
  podInfoOnMount: true
---
apiVersion: storage.k8s.io/v1beta1
kind: CSIDriver
metadata:
  name: nasplugin.csi.alibabacloud.com
spec:
  attachRequired: false
  podInfoOnMount: true
---
apiVersion: storage.k8s.io/v1beta1
kind: CSIDriver
metadata:
  name: ossplugin.csi.alibabacloud.com
spec:
  attachRequired: false
  podInfoOnMount: true
---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-plugin
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: csi-plugin
  template:
    metadata:
      labels:
        app: csi-plugin
        random.uuid: "{{ .UUID }}"
    spec:
      tolerations:
        - operator: Exists
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: type
                operator: NotIn
                values:
                - virtual-kubelet
      nodeSelector:
        beta.kubernetes.io/os: linux
      serviceAccount: csi-admin
      priorityClassName: system-node-critical
      hostNetwork: true
      hostPID: true
      containers:
        - name: disk-driver-registrar
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-node-driver-registrar:v1.3.0-6e9fff3-aliyun
          imagePullPolicy: Always
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
          args:
            - "--v=5"
            - "--csi-address=/var/lib/kubelet/csi-plugins/diskplugin.csi.alibabacloud.com/csi.sock"
            - "--kubelet-registration-path=/var/lib/kubelet/csi-plugins/diskplugin.csi.alibabacloud.com/csi.sock"
          volumeMounts:
            - name: kubelet-dir
              mountPath: /var/lib/kubelet
            - name: registration-dir
              mountPath: /registration
        - name: nas-driver-registrar
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-node-driver-registrar:v1.3.0-6e9fff3-aliyun
          imagePullPolicy: Always
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
          args:
            - "--v=5"
            - "--csi-address=/var/lib/kubelet/csi-plugins/nasplugin.csi.alibabacloud.com/csi.sock"
            - "--kubelet-registration-path=/var/lib/kubelet/csi-plugins/nasplugin.csi.alibabacloud.com/csi.sock"
          volumeMounts:
            - name: kubelet-dir
              mountPath: /var/lib/kubelet/
            - name: registration-dir
              mountPath: /registration
        - name: oss-driver-registrar
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-node-driver-registrar:v1.3.0-6e9fff3-aliyun
          imagePullPolicy: Always
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
          args:
            - "--v=5"
            - "--csi-address=/var/lib/kubelet/csi-plugins/ossplugin.csi.alibabacloud.com/csi.sock"
            - "--kubelet-registration-path=/var/lib/kubelet/csi-plugins/ossplugin.csi.alibabacloud.com/csi.sock"
          volumeMounts:
            - name: kubelet-dir
              mountPath: /var/lib/kubelet/
            - name: registration-dir
              mountPath: /registration
        - name: csi-plugin
          securityContext:
            privileged: true
            capabilities:
              add: ["SYS_ADMIN"]
            allowPrivilegeEscalation: true
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-plugin:{{.ImageVersion}}
          imagePullPolicy: "Always"
          args:
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=2"
            - "--driver=oss,nas,disk"
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
            - name: CSI_ENDPOINT
              value: unix://var/lib/kubelet/csi-plugins/driverplugin.csi.alibabacloud.com-replace/csi.sock
            - name: MAX_VOLUMES_PERNODE
              value: "15"
            - name: SERVICE_TYPE
              value: "plugin"
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          livenessProbe:
            httpGet:
              path: /healthz
              port: healthz
              scheme: HTTP
            initialDelaySeconds: 10
            periodSeconds: 30
            timeoutSeconds: 5
            failureThreshold: 5
          readinessProbe:
            httpGet:
              path: /healthz
              port: healthz
            initialDelaySeconds: 10
            periodSeconds: 30
            timeoutSeconds: 5
            failureThreshold: 5
          ports:
            - name: healthz
              containerPort: 11260
          volumeMounts:
            - name: kubelet-dir
              mountPath: /var/lib/kubelet/
              mountPropagation: "Bidirectional"
            - name: etc
              mountPath: /host/etc
            - name: host-log
              mountPath: /var/log/
            - name: ossconnectordir
              mountPath: /host/usr/
            - name: container-dir
              mountPath: /var/lib/container
              mountPropagation: "Bidirectional"
            - name: host-dev
              mountPath: /dev
              mountPropagation: "HostToContainer"
            - mountPath: /var/addon
              name: addon-token
              readOnly: true
      volumes:
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry
            type: DirectoryOrCreate
        - name: container-dir
          hostPath:
            path: /var/lib/container
            type: DirectoryOrCreate
        - name: kubelet-dir
          hostPath:
            path: /var/lib/kubelet
            type: Directory
        - name: host-dev
          hostPath:
            path: /dev
        - name: host-log
          hostPath:
            path: /var/log/
        - name: etc
          hostPath:
            path: /etc
        - name: ossconnectordir
          hostPath:
            path: /usr/
        - name: addon-token
          secret:
            defaultMode: 420
            optional: true
            items:
            - key: addon.token.config
              path: token-config
            secretName: addon.csi.token
  updateStrategy:
    rollingUpdate:
      maxUnavailable: 10%
    type: RollingUpdate
`


var csi_provision =`
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: alicloud-disk-available
provisioner: diskplugin.csi.alibabacloud.com
parameters:
    type: available
reclaimPolicy: Delete
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: alicloud-disk-essd
provisioner: diskplugin.csi.alibabacloud.com
parameters:
    type: cloud_essd
reclaimPolicy: Delete
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: alicloud-disk-ssd
provisioner: diskplugin.csi.alibabacloud.com
parameters:
    type: cloud_ssd
reclaimPolicy: Delete
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: alicloud-disk-efficiency
provisioner: diskplugin.csi.alibabacloud.com
parameters:
    type: cloud_efficiency
reclaimPolicy: Delete
allowVolumeExpansion: true
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: alicloud-disk-topology
provisioner: diskplugin.csi.alibabacloud.com
parameters:
    type: available
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
---
kind: Deployment
apiVersion: apps/v1
metadata:
  name: csi-provisioner
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: csi-provisioner
  strategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
    type: RollingUpdate
  replicas: 2
  template:
    metadata:
      labels:
        app: csi-provisioner
    spec:
      affinity:
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 1
            preference:
              matchExpressions:
              - key: node-role.kubernetes.io/master
                operator: Exists
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: type
                operator: NotIn
                values:
                - virtual-kubelet
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - csi-provisioner
            topologyKey: kubernetes.io/hostname
      tolerations:
      - effect: NoSchedule
        operator: Exists
        key: node-role.kubernetes.io/master
      - effect: NoSchedule
        operator: Exists
        key: node.cloudprovider.kubernetes.io/uninitialized
      serviceAccount: csi-admin
      priorityClassName: system-node-critical
      containers:
        - name: external-disk-provisioner
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-provisioner:v1.6.0-71838bd-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--provisioner=diskplugin.csi.alibabacloud.com"
            - "--csi-address=$(ADDRESS)"
            - "--feature-gates=Topology=True"
            - "--volume-name-prefix=disk"
            - "--strict-topology=true"
            - "--timeout=150s"
            - "--enable-leader-election=true"
            - "--leader-election-type=leases"
            - "--retry-interval-start=500ms"
            - "--v=5"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: disk-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com
        - name: external-disk-attacher
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-attacher:v2.1.0-b330d29-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election=true"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: disk-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com
        - name: external-disk-resizer
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-resizer:v1.1.0-7b30758-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: disk-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com
        - name: external-nas-provisioner
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-provisioner:v1.6.0-71838bd-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--provisioner=nasplugin.csi.alibabacloud.com"
            - "--csi-address=$(ADDRESS)"
            - "--volume-name-prefix=nas"
            - "--timeout=150s"
            - "--enable-leader-election=true"
            - "--leader-election-type=leases"
            - "--retry-interval-start=500ms"
            - "--v=5"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/csi-provisioner/nasplugin.csi.alibabacloud.com/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: nas-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/nasplugin.csi.alibabacloud.com
        - name: external-nas-resizer
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-resizer:v1.1.0-7b30758-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election"
          env:
            - name: ADDRESS
              value: /var/lib/kubelet/csi-provisioner/nasplugin.csi.alibabacloud.com/csi.sock
          imagePullPolicy: "Always"
          volumeMounts:
            - name: nas-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/nasplugin.csi.alibabacloud.com
        - name: external-csi-snapshotter
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-snapshotter:v4.0.0-5cbf27e-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--v=5"
            - "--csi-address=$(ADDRESS)"
            - "--leader-election=true"
            - "--extra-create-metadata=true"
          env:
            - name: ADDRESS
              value: /csi/csi.sock
          imagePullPolicy: Always
          volumeMounts:
            - name: disk-provisioner-dir
              mountPath: /csi
        - name: external-snapshot-controller
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/snapshot-controller:v4.0.0-5cbf27e-aliyun
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 500m
              memory: 1024Mi
          args:
            - "--v=5"
            - "--leader-election=true"
          imagePullPolicy: Always
        - name: csi-provisioner
          securityContext:
            privileged: true
          image: registry-vpc.{{.Region}}.aliyuncs.com/acs/csi-plugin:{{.ImageVersion}}
          imagePullPolicy: "Always"
          args:
            - "--endpoint=$(CSI_ENDPOINT)"
            - "--v=2"
            - "--driver=nas,disk"
          env:
            - name: CSI_ENDPOINT
              value: unix://var/lib/kubelet/csi-provisioner/driverplugin.csi.alibabacloud.com-replace/csi.sock
            - name: MAX_VOLUMES_PERNODE
              value: "15"
            - name: SERVICE_TYPE
              value: "provisioner"
            - name: "CLUSTER_ID"
              value: "empty.cluster.id"
          livenessProbe:
            httpGet:
              path: /healthz
              port: healthz
              scheme: HTTP
            initialDelaySeconds: 10
            periodSeconds: 30
            timeoutSeconds: 5
            failureThreshold: 5
          readinessProbe:
            httpGet:
              path: /healthz
              port: healthz
            initialDelaySeconds: 5
            periodSeconds: 20
          ports:
            - name: healthz
              containerPort: 11270
          volumeMounts:
            - name: host-log
              mountPath: /var/log/
            - name: disk-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/diskplugin.csi.alibabacloud.com
            - name: nas-provisioner-dir
              mountPath: /var/lib/kubelet/csi-provisioner/nasplugin.csi.alibabacloud.com
            - mountPath: /var/addon
              name: addon-token
              readOnly: true
          resources:
            limits:
              cpu: 500m
              memory: 1024Mi
            requests:
              cpu: 100m
              memory: 128Mi
      volumes:
        - name: disk-provisioner-dir
          emptyDir: {}
        - name: nas-provisioner-dir
          emptyDir: {}
        - name: host-log
          hostPath:
            path: /var/log/
        - name: addon-token
          secret:
            defaultMode: 420
            optional: true
            items:
            - key: addon.token.config
              path: token-config
            secretName: addon.csi.token
`


