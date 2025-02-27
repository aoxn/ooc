package addons

var TERWAY = ConfigTpl{
	Name:         "terway",
	Tpl:          terway,
	ImageVersion: "v1.2.1",
	IPStack:      "ipv4",
}

var terway = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: terway
  namespace: kube-system

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: terway-pod-reader
  namespace: kube-system
rules:
- apiGroups: [""]
  resources: ["pods", "nodes", "namespaces", "configmaps", "serviceaccounts", "endpoints", "services"]
  verbs: ["get", "watch", "list", "update", "patch"]
- apiGroups: [""]
  resources:
    - nodes
    - nodes/status
  verbs:
    - patch
- apiGroups: [ "coordination.k8s.io" ]
  resources: [ "leases" ]
  verbs: [ "get", "watch", "update", "create" ]
- apiGroups: [""]
  resources:
    - events
  verbs:
    - create
    - patch
- apiGroups: ["discovery.k8s.io"]
  resources:
    - endpointslices
  verbs:
    - get
    - list
    - watch
- apiGroups: ["networking.k8s.io"]
  resources:
  - networkpolicies
  verbs:
  - get
  - list
  - watch
- apiGroups: ["extensions"]
  resources:
  - networkpolicies
  verbs:
  - get
  - list
  - watch
- apiGroups: [""]
  resources:
  - pods/status
  verbs:
  - update
- apiGroups: ["crd.projectcalico.org"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - create
  - get
  - list
  - watch
  - update
- apiGroups:
  - cilium.io
  resources:
  - "*"
  verbs:
  - "*"
- apiGroups: [ "network.alibabacloud.com" ]
  resources: [ "*" ]
  verbs: [ "*" ]
---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: terway-binding
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: terway-pod-reader
subjects:
- kind: ServiceAccount
  name: terway
  namespace: kube-system

---
{{ if eq .Action "Install" }}
kind: ConfigMap
apiVersion: v1
metadata:
  name: eni-config
  namespace: kube-system
data:
  eni_conf: |
    {
      "version": "1",
      "max_pool_size": 5,
      "min_pool_size": 0,
      "vswitches": {{.PodVswitchId}},
      "eni_tags": {"wdrip.io":"orz"},
      "service_cidr": "{{.ServiceCIDR}}",
      "security_group": "{{.SecurityGroupID}}",
      "ip_stack": "{{.IPStack}}",
      "vswitch_selection_policy": "ordered"
    }
  10-terway.conf: |
    {
      "cniVersion": "0.3.1",
      "name": "terway",
      "type": "terway"
    }
  disable_network_policy: "false"
  in_cluster_loadbalance: "true"
{{ end }}
---

apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: terway-eniip
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: terway-eniip
  template:
    metadata:
      labels:
        app: terway-eniip
        random.uuid: "{{ .UUID }}"
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
    spec:
      hostPID: true
      priorityClassName: system-node-critical
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: type
                operator: NotIn
                values:
                - virtual-kubelet
              - key: beta.kubernetes.io/arch
                operator: In
                values:
                - amd64
                - arm64
              - key: beta.kubernetes.io/os
                operator: In
                values:
                - linux
            - matchExpressions:
              - key: type
                operator: NotIn
                values:
                - virtual-kubelet
              - key: kubernetes.io/arch
                operator: In
                values:
                - amd64
                - arm64
              - key: kubernetes.io/os
                operator: In
                values:
                - linux
      tolerations:
      - operator: "Exists"
      terminationGracePeriodSeconds: 0
      serviceAccountName: terway
      hostNetwork: true
      initContainers:
      - name: terway-init
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/terway:{{.ImageVersion}}
        imagePullPolicy: IfNotPresent
        securityContext:
          privileged: true
        command:
        - /bin/init.sh
        volumeMounts:
        - name: configvolume
          mountPath: /etc/eni
        - name: cni-bin
          mountPath: /opt/cni/bin/
        - name: cni
          mountPath: /etc/cni/net.d/
        - mountPath: /lib/modules
          name: lib-modules
        - mountPath: /host
          name: host-root
      containers:
      - name: terway
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/terway:{{.ImageVersion}}
        imagePullPolicy: IfNotPresent
        command: ["/usr/bin/terwayd", "-log-level", "info", "-daemon-mode", "ENIMultiIP"]
        securityContext:
          privileged: true
        resources:
          requests:
            cpu: "100m"
            memory: "100Mi"
          limits:
            cpu: "100m"
            memory: "256Mi"
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: POD_NAMESPACE
          valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        volumeMounts:
        - name: configvolume
          mountPath: /etc/eni
        - mountPath: /var/run/
          name: eni-run
        - mountPath: /opt/cni/bin/
          name: cni-bin
        - mountPath: /lib/modules
          name: lib-modules
        - mountPath: /var/lib/cni/networks
          name: cni-networks
        - mountPath: /var/lib/cni/terway
          name: cni-terway
        - mountPath: /var/lib/kubelet/device-plugins
          name: device-plugin-path
        - name: addon-token
          mountPath: "/var/addon"
          readOnly: true
      - name: policy
        image: registry-vpc.{{.Region}}.aliyuncs.com/acs/terway:{{.ImageVersion}}
        imagePullPolicy: IfNotPresent
        command: ["/bin/policyinit.sh"]
        env:
        - name: NODENAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: DISABLE_POLICY
          valueFrom:
            configMapKeyRef:
              name: eni-config
              key: disable_network_policy
              optional: true
        - name: FELIX_TYPHAK8SSERVICENAME
          valueFrom:
            configMapKeyRef:
              name: eni-config
              key: felix_relay_service
              optional: true
        - name: K8S_NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        - name: CILIUM_K8S_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        - name: CILIUM_CNI_CHAINING_MODE
          value: terway-chainer
        - name: IN_CLUSTER_LOADBALANCE
          valueFrom:
            configMapKeyRef:
              name: eni-config
              key: in_cluster_loadbalance
              optional: true
        securityContext:
          privileged: true
        resources:
          requests:
            cpu: "250m"
            memory: "100Mi"
          limits:
            cpu: "1"
        livenessProbe:
          httpGet: null
          tcpSocket:
            port: 9099
            host: localhost
          periodSeconds: 10
          initialDelaySeconds: 10
          failureThreshold: 6
        readinessProbe:
          httpGet: null
          tcpSocket:
            port: 9099
            host: localhost
          periodSeconds: 10
        volumeMounts:
        - mountPath: /lib/modules
          name: lib-modules
        - mountPath: /etc/cni/net.d
          name: cni
        - mountPath: /etc/eni
          name: configvolume
        # volumes use by cilium
        - mountPath: /sys/fs
          name: sys-fs
        - mountPath: /var/run/cilium
          name: cilium-run
        - mountPath: /host/opt/cni/bin
          name: cni-bin
        - mountPath: /host/etc/cni/net.d
          name: cni
          # Needed to be able to load kernel modules
        - mountPath: /run/xtables.lock
          name: xtables-lock
      volumes:
      - name: configvolume
        configMap:
          name: eni-config
          items:
          - key: eni_conf
            path: eni.json
          - key: 10-terway.conf
            path: 10-terway.conf
      - name: cni-bin
        hostPath:
          path: /opt/cni/bin
          type: "Directory"
      - name: cni
        hostPath:
          path: /etc/cni/net.d
      - name: eni-run
        hostPath:
          path: /var/run/
          type: "Directory"
      - name: lib-modules
        hostPath:
          path: /lib/modules
      - name: cni-networks
        hostPath:
          path: /var/lib/cni/networks
      - name: cni-terway
        hostPath:
          path: /var/lib/cni/terway
      - name: device-plugin-path
        hostPath:
          path: /var/lib/kubelet/device-plugins
          type: "Directory"
      - name: host-root
        hostPath:
          path: /
          type: "Directory"
      - name: addon-token
        secret:
          secretName: addon.network.token
          items:
          - key: addon.token.config
            path: token-config
          optional: true
      # used by cilium
      # To keep state between restarts / upgrades
      - hostPath:
          path: /var/run/cilium
          type: DirectoryOrCreate
        name: cilium-run
      # To keep state between restarts / upgrades for bpf maps
      - hostPath:
          path: /sys/fs/
          type: DirectoryOrCreate
        name: sys-fs
        # To access iptables concurrently with other processes (e.g. kube-proxy)
      - hostPath:
          path: /run/xtables.lock
          type: FileOrCreate
        name: xtables-lock

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: felixconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: FelixConfiguration
    plural: felixconfigurations
    singular: felixconfiguration

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: bgpconfigurations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: BGPConfiguration
    plural: bgpconfigurations
    singular: bgpconfiguration

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: ippools.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: IPPool
    plural: ippools
    singular: ippool

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: hostendpoints.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: HostEndpoint
    plural: hostendpoints
    singular: hostendpoint

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: clusterinformations.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: ClusterInformation
    plural: clusterinformations
    singular: clusterinformation

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: globalnetworkpolicies.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: GlobalNetworkPolicy
    plural: globalnetworkpolicies
    singular: globalnetworkpolicy

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: globalnetworksets.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: GlobalNetworkSet
    plural: globalnetworksets
    singular: globalnetworkset

---

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: networkpolicies.crd.projectcalico.org
spec:
  scope: Namespaced
  group: crd.projectcalico.org
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
  names:
    kind: NetworkPolicy
    plural: networkpolicies
    singular: networkpolicy
`
