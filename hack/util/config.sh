#!/bin/bash

cat > $CUR_DIR/config.txt << EOF
clusterid: "${CLUSTER_NAME}"
iaas:
  provider:
    name: dev
    value:
      region: "${REGION}"
      accessKey: "${ACCESS_KEY_ID}"
      accessSecret: "${ACCESS_KEY_SECRET}"
      template: ${SRC_DIR}/pkg/iaas/provider/dev/demo.dev.json
  workerCount: 1
  image: "${IMAGE_ID}"
  disk:
    size: 40G
    type: "${DISK_TYPE}"
  region: "${REGION}"
  zoneid: ${REGION}-k
  instance: "${INSTANCE_TYPE}"
registry: registry-vpc.${REGION}.aliyuncs.com
namespace: default
cloudType: public
kubernetes:
  name: kubernetes
  version: 1.16.9-aliyun.1
  kubeadmToken: 8rkjd9.8e5ruau8rsc3utex
etcd:
  name: etcd
  version: v3.4.3
runtime:
  name: docker
  version: 19.03.5
  para:
    key1: value
    key2: value2
sans:
  - 192.168.0.1
network:
  mode: ipvs
  podcidr: 172.16.0.1/16
  svccidr: 172.19.0.1/20
  domain: cluster.domain
  netMask: 25
endpoint:
  intranet: "${INTRANET_LB}"
  internet: "${INTERNET_LB}"
EOF
