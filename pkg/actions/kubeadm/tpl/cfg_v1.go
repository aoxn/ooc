package tpl

var Tplv1 = `
apiVersion: kubeadm.k8s.io/v1beta1
kind: InitConfiguration
metadata:
  name: "initconfig"
bootstrapTokens:
- token: "{{ .Kubernetes.KubeadmToken }}"
  ttl: 0s
nodeRegistration:
  kubeletExtraArgs:
    cloud-provider: external
  name: "{{ .NodeName }}"
---
apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: "clusterconfig"
apiServer:
  extraArgs:
    profiling: "false"
    cloud-provider: external
    #service-node-port-range: "$NODE_PORT_RANGE"
    feature-gates: VolumeSnapshotDataSource=true,CSINodeInfo=true,CSIDriverRegistry=true
  extraVolumes:
  - hostPath: /etc/localtime
    mountPath: /etc/localtime
    name: localtime
  certSANs:
{{ range $_, $v := .Sans }}
  - {{$v}} 
{{ end }}
  - 127.0.0.1
controllerManager:
  extraArgs:
    profiling: "false"
    cloud-provider: external
    horizontal-pod-autoscaler-use-rest-clients: "true"
{{ if .Network.NetMask }}
    node-cidr-mask-size: "{{ .Network.NetMask }}"
{{ end }}
  extraVolumes:
  - hostPath: /etc/localtime
    mountPath: /etc/localtime
    name: localtime
scheduler:
  extraArgs:
    profiling: "false"
  extraVolumes:
  - hostPath: /etc/localtime
    mountPath: /etc/localtime
    name: localtime
etcd:
  external:
    caFile: /var/lib/etcd/cert/server-ca.crt
    certFile: /var/lib/etcd/cert/client.crt
    keyFile: /var/lib/etcd/cert/client.key
    endpoints:
{{ range $_, $v := .EtcdEndpoints }}
    - {{$v}} 
{{ end }}
networking:
  podSubnet: "{{ .Network.PodCIDR }}"
  dnsDomain: {{ .Network.Domain }}
  serviceSubnet: {{ .Network.SVCCIDR }}
imageRepository: {{ .Registry }}
kubernetesVersion: v{{ .Kubernetes.Version }}
clusterName: "{{ .ClusterID }}"
`
