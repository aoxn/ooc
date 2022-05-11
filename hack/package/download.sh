#!/bin/bash

ARCH=amd64
OS=centos
VERSION=v1.20.4-aliyun.1
FILE_SERVER=http://aliacs-k8s-cn-hangzhou.oss.aliyuncs.com

function download() {
	local mdir=.wdrip/cache/wdrip/default/public/kubernetes/$VERSION/$ARCH/$OS/bin/
	mkdir -p "$mdir"
	pkgs=(
		kubeadm
		kubectl
		kubelet
		crictl
	)
	for pkg in ${pkgs};
	do
		src="$FILE_SERVER"/bin/${VERSION}/linux/$ARCH/$pkg
		if [[ "$pkg" == "crictl" ]];
		then
			src=$FILE_SERVER/pkg/common/kubernetes/1.11.2/bin/crictl
		fi
		wget -O $mdir/$pkg.unzip "$src"; chmod +x $mdir/$pkg.unzip
		~/vaoxn/code/wdrip/build/tool/upx.darwin -9 -o $mdir/$pkg $mdir/$pkg.unzip
	done
}

download

