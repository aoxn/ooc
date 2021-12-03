#!/usr/bin/env bash

set -x -e
# PATH: ${PKG_FILE_SERVER}${NAMESPACE}/public/run/2.0/run.replace.sh


########################
#
#  Author: aoxn
#  Usage:
#     export ROLE=WORKER TOKEN=aacdeb.61c3abfd1eac6pbc INTRANET_LB=172.16.9.43; bash run.replace.sh
########################

# change to /root/ dir
WORKDIR=/root/
cd ${WORKDIR}

function detecos() {
    export OS=centos
    export ARCH=amd64
}

function validatedefault() {
    # detect os arch
    detecos
    if [[ "$PKG_BUCKET" == "" ]];
    then
        PKG_BUCKET="host-ovm"
        echo "using oss bucket [oss://$PKG_BUCKET-$REGION] as package file server"
    fi
    if [[ "$BOOT_TYPE" == "" ]];
    then
        BOOT_TYPE="local"
    fi

    if [[ "$REGION" == "" ]];
    then
        # https://github.com/koalaman/shellcheck/wiki/SC2155
        REGION="$(curl 100.100.100.200/latest/meta-data/region-id)"; export REGION
    fi

    if [[ "$ROLE" == "" ]];
    then
        echo "ROLE must be provided, one of BOOTSTRAP|MASTER|WORKER"; exit 1
    fi
    if [[ "$NAMESPACE" == "" ]];
    then
        NAMESPACE=default
    fi
    if [[ "$OVM_VERSION" == "" ]];
    then
        export OVM_VERSION=0.1.1
    fi

    if [[ "$CLOUD_TYPE" == "" ]];
    then
        export CLOUD_TYPE=public
    fi

    if [[ "$TOKEN" == "" ]];
    then
        echo "TOKEN must be provided"; exit 1
    fi

    if [[ "$PKG_FILE_SERVER" == "" ]];
    then
        PKG_FILE_SERVER="http://${PKG_BUCKET}-$REGION.oss-$REGION-internal.aliyuncs.com"
        echo "empty PKG_FILE_SERVER, using default ${PKG_FILE_SERVER}"
    fi
    export BIN_PATH=/usr/local/bin/
    echo "using beta version: [${NAMESPACE}]"
    wget --tries 10 --no-check-certificate -q \
        -O /tmp/ovm.${ARCH}\
        "${PKG_FILE_SERVER}"/ovm/${NAMESPACE}/${CLOUD_TYPE}/ovm/${OVM_VERSION}/${ARCH}/${OS}/ovm.${ARCH}
    chmod +x /tmp/ovm.${ARCH} ; mv /tmp/ovm.${ARCH} $BIN_PATH/ovm
}

function bootstrap() {
    echo run bootstrap init
    # run bootsrap init
    nohup ovm bootstrap --token "${TOKEN}" --bootcfg /etc/ovm/ovm.cfg &
}

function init() {
    echo run master init
    case $BOOT_TYPE in
    "local")
      # run master init
      ovm init --role "${ROLE}" --token "${TOKEN}" --config /etc/ovm/ovm.cfg
      ;;
    "operator")
      # run master init
      ovm init --role "${ROLE}" --token "${TOKEN}" --boot-type "${BOOT_TYPE}" --endpoint "${ENDPOINT}"
      ;;
    esac
}

function join() {
    echo run worker init
    # run master init
    if [[ "$ENDPOINT" == "" ]];
    then
        echo "endpoint must be specified with env"; exit 1
    fi
    ovm init --role Worker --token "${TOKEN}" --endpoint "${ENDPOINT}"  --boot-type operator
}

function postcheck() {

    echo 'Check ROS notify server health, and notify to ROS notify server if its healthy.'
    set +e
    for ((i=1; i<=5; i ++));
    do
        cnt=$(curl -s http://100.100.100.110/health-check | grep ok | wc -l)
        echo "wait for ros notify server to be healthy cnt=$cnt, this is round $i"
        if curl -s http://100.100.100.110/health-check | grep ok ;
        then
            echo "the ros notify server is healthy"; break
        fi
        sleep 2
    done
    if ! curl -s http://100.100.100.110/health-check | grep ok ;
    then
        echo "wait for ros notify server to be healthy failed."; exit 2
    fi
    set -e
}

# validate default parameter first
validatedefault
#config

case ${ROLE} in
    "Hybrid")
        echo "join master"
        init
    ;;
    "Master")
        echo "join master"
        init
    ;;
    "Worker")
        echo "join worker"
        join
    ;;
    "Bootstrap")
        echo "bootstrap master"
        bootstrap; init
    ;;
    *)
        echo "unrecognized role"
    ;;
esac

postcheck
