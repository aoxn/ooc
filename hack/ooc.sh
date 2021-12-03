#!/bin/bash

export CUR_DIR=$(cd "$(dirname "$0")";pwd)
export SRC_DIR=$(dirname "$CUR_DIR")

export CLUSTER_NAME=kubernetes-ooc-64
export ACCESS_KEY_ID=
export ACCESS_KEY_SECRET=

export INSTANCE_TYPE=ecs.c6.xlarge
export REGION=cn-hangzhou
export IMAGE_ID=centos_7_9_x64_20G_alibase_20210623.vhd
export DIST_TYPE=cloud_essd

source $CUR_DIR/util/config.sh

function create_ooc_cluster() {
  me=$CUR_DIR/config.txt
  build/bin/ooc create --config ${me}
}

ACTION=$1;shift

if [[ "$ACTION" == "" ]];
then
  ACTION="get"
  echo "没有指定操作，默认[get]"
fi

case $ACTION in
create)
  create_ooc_cluster
  ;;
get)
  build/bin/ooc get
  ;;
delete)
  build/bin/ooc delete $@
  ;;
watch)
  build/bin/ooc watch --name "$CLUSTER_NAME"
  ;;
config)
  build/bin/ooc kubeconfig --name "$CLUSTER_NAME" >~/.kube/config.ooc
  ;;
esac

# clean up
rm -rf $CUR_DIR/config.txt