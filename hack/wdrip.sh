#!/bin/bash

export CUR_DIR=$(cd "$(dirname "$0")";pwd)
export SRC_DIR=$(dirname "$CUR_DIR")

export CLUSTER_NAME=kubernetes-wdrip-124

export INSTANCE_TYPE=ecs.c6.xlarge
export REGION=cn-hangzhou
export IMAGE_ID=centos_7_9_x64_20G_alibase_20210623.vhd
export DIST_TYPE=cloud_essd

export Registry=registry-vpc.cn-hangzhou.aliyuncs.com/aoxn
export Version=0.0.1-g3e8f84b

source $CUR_DIR/util/config.sh
source $CUR_DIR/util/chaos.monkey.sh

eval "$(cat ~/.security/ak.wdrip)" || true

if [[ "$ACCESS_KEY_ID" == "" ]];
then
  echo "ACCESS_KEY_ID & ACCESS_KEY_SECRET must be set"; exit 1
fi

function create_wdrip_cluster() {
  me=$CUR_DIR/config.txt
  build/bin/wdrip create --config ${me}
}

ACTION=$1;shift

if [[ "$ACTION" == "" ]];
then
  ACTION="get"
  echo "没有指定操作，默认[get]"
fi

case $ACTION in
create)
  create_wdrip_cluster
  ;;
get)
  build/bin/wdrip get
  ;;
delete)
  build/bin/wdrip delete $@
  ;;
watch)
  build/bin/wdrip watch --name "$CLUSTER_NAME"
  ;;
config)
  build/bin/wdrip kubeconfig --name "$CLUSTER_NAME" >~/.kube/config.wdrip
  ;;
chaos)
  echo "chaos monkey"
  kubectl --kubeconfig ~/.kube/config.wdrip apply -f hack/config.monkey.txt
  ;;
esac

# clean up
rm -rf $CUR_DIR/config.txt
