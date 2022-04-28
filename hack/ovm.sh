#!/bin/bash

export CUR_DIR=$(cd "$(dirname "$0")";pwd)
export SRC_DIR=$(dirname "$CUR_DIR")

export CLUSTER_NAME=kubernetes-ovm-121

export INSTANCE_TYPE=ecs.c6.xlarge
export REGION=cn-hangzhou
export IMAGE_ID=centos_7_9_x64_20G_alibase_20210623.vhd
export DIST_TYPE=cloud_essd

export Registry=registry-vpc.cn-hangzhou.aliyuncs.com/aoxn
export Version=0.0.1-g3e8f84b

source $CUR_DIR/util/config.sh
source $CUR_DIR/util/chaos.monkey.sh

eval "$(cat ~/.security/ak.ovm)" || true

if [[ "$ACCESS_KEY_ID" == "" ]];
then
  echo "ACCESS_KEY_ID & ACCESS_KEY_SECRET must be set"; exit 1
fi

function create_ovm_cluster() {
  me=$CUR_DIR/config.txt
  build/bin/ovm create --config ${me}
}

ACTION=$1;shift

if [[ "$ACTION" == "" ]];
then
  ACTION="get"
  echo "没有指定操作，默认[get]"
fi

case $ACTION in
create)
  create_ovm_cluster
  ;;
get)
  build/bin/ovm get
  ;;
delete)
  build/bin/ovm delete $@
  ;;
watch)
  build/bin/ovm watch --name "$CLUSTER_NAME"
  ;;
config)
  build/bin/ovm kubeconfig --name "$CLUSTER_NAME" >~/.kube/config.ovm
  ;;
chaos)
  echo "chaos monkey"
  kubectl --kubeconfig ~/.kube/config.ovm apply -f hack/config.monkey.txt
  ;;
esac

# clean up
rm -rf $CUR_DIR/config.txt
