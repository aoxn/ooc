


REPOSITORY="https://host-ovm-cn-hangzhou.oss-cn-hangzhou.aliyuncs.com"

VERSION=0.1.1

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
    URL="${REPOSITORY}/ovm/default/public/ovm/${VERSION}/darwin/macos/ovm"
    ;;
  "linux")
    URL="ovm/default/public/ovm/${VERSION}/$ARCH/centos/ovm.${ARCH}"
    ;;
  esac
  echo "trying to get ovm from[${URL}]"
  wget -t 3 -q -O ovm "$URL" && chmod +x ovm && mv ovm "$dst"

  # config ovm context
  config
  echo "install finished"
}

function config() {
  cach=~/.ovm
  mkdir -p ~/.ovm/
  if [[ -f $cach/config ]];
  then
    echo "ovm config context is already exist, skip configuration"
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
      bucketName: ovm-index
      region: cn-hangzhou
EOF
}

install

# ossutil --endpoint cn-hangzhou.oss.aliyuncs.com cp -u cmd/ovm/build/install.sh oss://host-ovm-cn-hangzhou/ovm/install.sh

# curl -sSL --retry 3 https://host-ovm-cn-hangzhou.oss-cn-hangzhou.aliyuncs.com/ovm/install.sh






