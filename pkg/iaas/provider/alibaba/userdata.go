package alibaba

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"text/template"
)

func PrefixPart() string {
	return `#!/bin/sh
set -x -e
`
}

func NewUserData(ctx *provider.Context) string {
	cfg := ctx.BootCFG()
	cfg.Namespace = "default"
	cfg.Registry = fmt.Sprintf("registry-vpc.%s.aliyuncs.com/acs", cfg.Bind.Region)
	if cfg.Endpoint.Intranet == "" {
		cfg.Endpoint.Intranet = "${INTRANET_LB}"
	}
	if cfg.Endpoint.Internet == "" {
		cfg.Endpoint.Internet = "${INTERNET_LB}"
	}
	return fmt.Sprintf(
		USER_DATA,
		"Hybrid", cfg.Kubernetes.KubeadmToken,
		cfg.CloudType, cfg.Namespace, utils.PrettyYaml(cfg), cfg.Bind.Provider.Name,
	)
}

var USER_DATA = `
REGION="$(curl 100.100.100.200/latest/meta-data/region-id)"
export REGION
export ROLE=%s OS=centos ARCH=amd64 \
	   TOKEN=%s \
       CLOUD_TYPE=%s \
       NAMESPACE=%s \
	   FILE_SERVER="http://host-oc-$REGION.oss-$REGION-internal.aliyuncs.com"
echo "using beta version: [${NAMESPACE}]"
mkdir -p /etc/ooc;
echo "
%s
" > /etc/ooc/ooc.cfg
wget --tries 10 --no-check-certificate -q \
     -O run.replace.sh\
     ${FILE_SERVER}/ack/${NAMESPACE}/${CLOUD_TYPE}/run/2.0/${ARCH}/${OS}/run.%s.sh
time bash run.replace.sh |tee /var/log/init.log
`

type ConfigTpl struct {
	Namespace  string
	Token      string
	OOCVersion string
	Endpoint   string
	Role       string
	RunVersion string
	CloudType  string
	Arch       string
	OS         string
	Provider   string
}

func NewWorkerUserData(ctx *provider.Context) (string, error) {
	boot := ctx.BootCFG()
	cfg := &ConfigTpl{
		Namespace:  boot.Namespace,
		Token:      boot.Kubernetes.KubeadmToken,
		OOCVersion: "0.1.1",
		Endpoint:   fmt.Sprintf("http://%s:9443", boot.Endpoint.Intranet),
		Role:       "Worker",
		RunVersion: "2.0",
		CloudType:  "public",
		Arch:       "amd64",
		OS:         "centos",
		Provider:   "alibaba",
	}
	tpl, err := template.New("userdata").Parse(WorkUserData)
	if err != nil {
		return "", errors.Wrap(err, "build worker userdata")
	}
	out := bytes.NewBufferString("")
	err = tpl.Execute(out, cfg)
	if err != nil {
		return "", errors.Wrap(err, "parse userdata")
	}
	klog.Infof("DEBUG, worker userdata: \n%s", out.String())
	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

var WorkUserData = `#!/bin/sh
set -x -e
REGION=$(curl --retry 5 -sSL http://100.100.100.200/latest/meta-data/region-id)
export REGION
export NAMESPACE={{ .Namespace }} \
       TOKEN={{ .Token }} \
       OOC_VERSION={{ .OOCVersion }} \
       FILE_SERVER=http://host-oc-${REGION}.oss-${REGION}-internal.aliyuncs.com \
       ENDPOINT={{ .Endpoint }} \
       ROLE={{ .Role }}
wget --tries 10 --no-check-certificate -q \
     -O run.replace.sh\
     ${FILE_SERVER}/ack/${NAMESPACE}/{{ .CloudType }}/run/{{ .RunVersion }}/{{.Arch }}/{{ .OS }}/run.{{ .Provider }}.sh
time bash run.replace.sh |tee /var/log/init.log

`

func NewRecoverUserData(ctx *provider.Context) (string, error) {
	boot := ctx.BootCFG()
	cfg := &ConfigTpl{
		Namespace:  boot.Namespace,
		Token:      boot.Kubernetes.KubeadmToken,
		OOCVersion: "0.1.1",
		Endpoint:   fmt.Sprintf("http://%s:9443", boot.Endpoint.Intranet),
		Role:       "Worker",
		RunVersion: "2.0",
		CloudType:  "public",
		Arch:       "amd64",
		OS:         "centos",
		Provider:   "alibaba",
	}
	ctxCfg := provider.BuildContexCFG(boot)
	me := struct {
		ConfigTpl
		ClusterName string
		OocConfig   string
	}{
		ConfigTpl:   *cfg,
		OocConfig:   utils.PrettyYaml(ctxCfg),
		ClusterName: boot.ClusterID,
	}
	tpl, err := template.New("restore userdata").Parse(RecoverUserData)
	if err != nil {
		return "", errors.Wrap(err, "build recover userdata")
	}
	out := bytes.NewBufferString("")
	err = tpl.Execute(out, me)
	if err != nil {
		return "", errors.Wrap(err, "parse recover userdata")
	}
	klog.Infof("DEBUG, recover userdata: \n%s", out.String())
	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

var RecoverUserData = `#!/bin/bash
set -e -x
REGION="$(curl 100.100.100.200/latest/meta-data/region-id)"
export REGION
export ROLE={{ .Role }} OS={{ .OS }} ARCH={{ .Arch }} \
       TOKEN={{ .Token }} \
       CLOUD_TYPE={{ .CloudType }} \
       NAMESPACE={{ .Namespace }} \
       OOC_VERSION={{ .OOCVersion }} \
       FILE_SERVER="http://host-oc-$REGION.oss-$REGION-internal.aliyuncs.com"
# set ooc operator endpoint
export ENDPOINT={{ .Endpoint }}
echo "using beta version: [${NAMESPACE}]"

echo "using beta version: [${NAMESPACE}]"
wget --tries 10 --no-check-certificate -q \
	-O /tmp/ooc.${ARCH}\
	"${FILE_SERVER}"/ack/${NAMESPACE}/${CLOUD_TYPE}/ooc/${OOC_VERSION}/${ARCH}/${OS}/ooc.${ARCH}
chmod +x /tmp/ooc.${ARCH} ; mv /tmp/ooc.${ARCH} /usr/local/bin/ooc; mkdir -p ~/.ooc/
cat > ~/.ooc/config << EOF
{{ .OocConfig }}
EOF
/usr/local/bin/ooc recover --recover-mode node --name {{ .ClusterName }}
`

var USER_DATA_JOIN_MASTER = `#!/bin/sh
set -e -x
REGION="$(curl 100.100.100.200/latest/meta-data/region-id)"
export REGION
# make sure ooc boot master from operator
export BOOT_TYPE=operator
export ROLE={{ .Role }} OS={{ .OS }} ARCH={{ .Arch }} \
       TOKEN={{ .Token }} \
       CLOUD_TYPE={{ .CloudType }} \
       NAMESPACE={{ .Namespace }} \
       FILE_SERVER="http://host-oc-$REGION.oss-$REGION-internal.aliyuncs.com"
# set ooc operator endpoint
export ENDPOINT={{ .Endpoint }}
echo "using beta version: [${NAMESPACE}]"
mkdir -p /etc/ooc;
wget --tries 10 --no-check-certificate -q \
     -O run.sh\
     ${FILE_SERVER}/ack/${NAMESPACE}/${CLOUD_TYPE}/run/2.0/${ARCH}/${OS}/run.{{.Provider}}.sh
time bash run.sh |tee /var/log/init.log.ack
`

func NewJoinMasterUserData(
	ctx *provider.Context,
) (string, error) {
	boot := ctx.BootCFG()
	cfg := &ConfigTpl{
		Namespace:  boot.Namespace,
		Token:      boot.Kubernetes.KubeadmToken,
		OOCVersion: "0.1.1",
		Endpoint:   fmt.Sprintf("http://%s:9443", boot.Endpoint.Intranet),
		Role:       "Hybrid",
		RunVersion: "2.0",
		CloudType:  "public",
		Arch:       "amd64",
		OS:         "centos",
		Provider:   "alibaba",
	}
	tpl, err := template.New("joinmaster").Parse(USER_DATA_JOIN_MASTER)
	if err != nil {
		return "", errors.Wrap(err, "build join master userdata")
	}
	out := bytes.NewBufferString("")
	err = tpl.Execute(out, cfg)
	if err != nil {
		return "", errors.Wrap(err, "parse join master userdata")
	}
	klog.Infof("DEBUG, join master userdata: \n%s", out.String())
	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}
