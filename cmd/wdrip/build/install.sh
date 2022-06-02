#!/bin/bash
set -eE

REPOSITORY="https://host-wdrip-cn-hangzhou.oss-cn-hangzhou.aliyuncs.com"

VERSION=0.1.1

errormsg() {
      echo install failed
}
trap errormsg ERR

# shellcheck disable=SC2005
# shellcheck disable=SC2021
export OS=$(echo "$(uname)"|tr '[A-Z]' '[a-z]')

function arch() {
        export ARCH=amd64
        case $(uname -m) in
                "x86-64")
                        export ARCH=amd64
                        ;;
                "arm64")
                        export ARCH=arm64
                        ;;
                "aarch64")
                        export ARCH=arm64
                        ;;
        esac
        echo "use ARCH=${ARCH} OS=${OS}"
}
arch

function install() {
        dst=/usr/local/bin/
        mkdir -p "$dst"
        case $OS in
        "darwin")
                URL="${REPOSITORY}/wdrip/default/public/wdrip/${VERSION}/darwin/macos/wdrip"
                ;;
        "linux")
                URL="${REPOSITORY}/wdrip/default/public/wdrip/${VERSION}/$ARCH/centos/wdrip.${ARCH}"
                ;;
        esac
        echo "trying to get wdrip from[${URL}]"
        if ! wget -t 3 -q -O wdrip "$URL";
        then
                echo "download wdrip failed!"; exit 1
        fi
        chmod +x wdrip && mv wdrip "$dst"

        # config wdrip context
        config
        echo "install finished"
}

function config() {
        echo "config local default provider "
        cach=~/.wdrip
        mkdir -p ~/.wdrip/
        if [[ -f $cach/config ]];
        then
                echo "wdrip config context is already exist, skip configuration"
                return
        fi
        cat > $cach/config <<EOF
apiVersion: alibabacloud.com/v1
contexts:
- context:
                provider-key: alibaba.dev
        name: devEnv
current-context: devEnv
kind: Config
providers:
- name: alibaba.dev
        provider:
                name: alibaba
                value:
                        accessKeyId: {replace-with-your-own-accessKeyId}
                        accessKeySecret: {replace-with-your-own-accessKeySecret}
                        bucketName: wdrip-index
                        region: cn-hangzhou
EOF
        echo "local provider config finished"
}

install

# ossutil --endpoint cn-hangzhou.oss.aliyuncs.com cp -u cmd/wdrip/build/install.sh oss://host-wdrip-cn-hangzhou/wdrip/install.sh

# curl -sSL --retry 3 https://host-wdrip-cn-hangzhou.oss-cn-hangzhou.aliyuncs.com/wdrip/install.sh






