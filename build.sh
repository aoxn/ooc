#!/usr/bin/env bash

set -e -x

if [[ "$BUILD" == "" ]];
then
    BUILD=jenkins
fi

if [[ "$OS" == "" ]];
then
    OS=centos
fi

if [[ "$VERSION" == "" ]];
then
    VERSION=0.1.0
fi

if [[ "$NAMESPACE" != "" ]];
then
    NAMESPACE=/${NAMESPACE}
else
    NAMESPACE=/aoxn
fi

if [[ "$REGIONS" == "" ]];
then
    REGIONS=(
        cn-hangzhou
    )
fi

if [[ "$CLOUD_TYPE" == "" ]];
then
    export CLOUD_TYPE=/public
else
    export CLOUD_TYPE=/${CLOUD_TYPE}
fi

function makeimage() {

    # build & push image
    make push

    osscmd put bin/ooc oss://host-oc-cn-hangzhou/ack${NAMESPACE}${CLOUD_TYPE}/ooc/${VERSION}/amd64/linux/bin/ooc
}

function makebinamd64() {
    make booc
    for region in "${REGIONS[@]}";
    do
        osscmd put bin/ooc oss://host-oc-${region}/ack${NAMESPACE}${CLOUD_TYPE}/ooc/${VERSION}/amd64/linux/bin/ooc
    done
}

function pushrun() {

    for region in "${REGIONS[@]}";
    do
        ossutil --endpoint  put deploy/run.sh oss://host-oc/ack${NAMESPACE}${CLOUD_TYPE}/run/2.0/${OS}/run.sh
    done
}

function makeimage() {
    mkdir -p bin/
    osscmd get oss://host-oc-cn-hangzhou/ack${NAMESPACE}${CLOUD_TYPE}/ooc/${VERSION}/amd64/linux/bin/ooc bin/ooc
    chmod +x bin/ooc
    image=registry.cn-hangzhou.aliyuncs.com/acs/ooc:`bin/ooc version`
    docker build -t ${image} .
    for region in "${REGIONS[@]}";
    do
        if docker images|awk '{print $1":"$2}'|grep ooc:`bin/ooc version`;
        then
            docker tag ${image} registry.${region}.aliyuncs.com/acs/ooc:`bin/ooc version`
        fi
        docker push registry.${region}.aliyuncs.com/acs/ooc:`bin/ooc version`
    done
}

case ${BUILD} in
    simple)
        makebinamd64 ; pushrun
    ;;
    jenkins)
        pushrun
    ;;
    image)
        makeimage
    ;;
esac