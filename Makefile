# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Simple makefile to build ovm quickly and reproducibly in a container
# Only requires docker on the host

OS_TYPE :=$(shell echo `uname`|tr '[A-Z]' '[a-z]')

# settings
REPO_ROOT:=${CURDIR}
# autodetect host GOOS and GOARCH by default, even if go is not installed
GOOS?=$(shell hack/util/goos.sh)
GOARCH?=$(shell hack/util/goarch.sh)
REGISTRY=registry.cn-hangzhou.aliyuncs.com/aoxn/ovm
TAG?=$(shell hack/util/tag.sh)

# make install will place binaries here
# the default path attempst to mimic go install
INSTALL_DIR?=$(shell hack/util/goinstalldir.sh)

# the output binary name, overridden when cross compiling
KIND_BINARY_NAME?=ovm
# use the official module proxy by default
GOPROXY?=https://mirrors.aliyun.com/goproxy
# default build image
GO_VERSION?=1.14.3
GO_IMAGE?=golang:$(GO_VERSION)
# docker volume name, used as a go module / build cache
CACHE_VOLUME?=ovm-build-cache

# variables for consistent logic, don't override these
CONTAINER_REPO_DIR=/src/ovm
CONTAINER_OUT_DIR=$(CONTAINER_REPO_DIR)/bin
OUT_DIR=$(REPO_ROOT)/build/bin
UID:=$(shell id -u)
GID:=$(shell id -g)

# standard "make" target -> builds
all: build

# creates the cache volume
make-cache:
	@echo + Ensuring build cache volume exists
	docker volume create $(CACHE_VOLUME)

# cleans the cache volume
clean-cache:
	@echo + Removing build cache volume
	docker volume rm $(CACHE_VOLUME)

# creates the output directory
out-dir:
	@echo + Ensuring build output directory exists
	mkdir -p $(OUT_DIR)

# cleans the output directory
clean-output:
	@echo + Removing build output directory
	rm -rf $(OUT_DIR)/

# builds ovm in a container, outputs to $(OUT_DIR)
ovm: make-cache out-dir
	@echo + Building ovm binary
	docker run \
		--rm \
		-v $(CACHE_VOLUME):/go \
		-e GOCACHE=/go/cache \
		-v $(OUT_DIR):/out \
		-v $(REPO_ROOT):$(CONTAINER_REPO_DIR) \
		-w $(CONTAINER_REPO_DIR) \
		-e GO111MODULE=on \
		-e GOPROXY=$(GOPROXY) \
		-e CGO_ENABLED=0 \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-e HTTP_PROXY=$(HTTP_PROXY) \
		-e HTTPS_PROXY=$(HTTPS_PROXY) \
		-e NO_PROXY=$(NO_PROXY) \
		--user $(UID):$(GID) \
		$(GO_IMAGE) \
		go build -v -o /out/$(KIND_BINARY_NAME).unzip \
		    -ldflags "-X github.com/aoxn/ovm.Version=$(TAG) -s -w" .
	@echo + Built ovm binary to $(OUT_DIR)/$(KIND_BINARY_NAME).unzip
	rm -f build/bin/ovm
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/ovm $(OUT_DIR)/$(KIND_BINARY_NAME).unzip
	rm -f $(OUT_DIR)/$(KIND_BINARY_NAME).unzip

# alias for building ovm
buildovm: ovm


image:
	docker build -t $(REGISTRY):$(TAG) build/

bimage:
	docker build -t $(REGISTRY):$(TAG) build/

push: bimage
	docker push $(REGISTRY):$(TAG)

# use: make install INSTALL_DIR=/usr/local/bin
install: build
	@echo + Copying ovm binary to INSTALL_DIR
	install $(OUT_DIR)/$(KIND_BINARY_NAME) $(INSTALL_DIR)/$(KIND_BINARY_NAME)


#-X gitlab.alibaba-inc.com/cos/ros.Template=$(ROS_TPL)
ovmmac:
	GOARCH=amd64 \
	GOOS=darwin \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ovm.unzip \
	-ldflags "-X github.com/aoxn/ovm.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built ovm binary to build/bin/ovm
	rm -f build/bin/ovm
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/ovm build/bin/ovm.unzip
	rm -f build/bin/ovm.unzip

ovmlinux:
	GOARCH=amd64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ovm.amd64.unzip \
	-ldflags "-X github.com/aoxn/ovm.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built ovm binary to build/bin/ovm.amd64
	rm -f build/bin/ovm.amd64
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/ovm.amd64 build/bin/ovm.amd64.unzip
	rm -f build/bin/ovm.amd64.unzip

ovmwin:
	GOARCH=amd64 \
	GOOS=windows \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ovm.exe \
       -ldflags "-X github.com/aoxn/ovm.Version=$(TAG) -s -w" cmd/main.go
ovmarm64:
	GOARCH=arm64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ovm.arm64.unzip \
	-ldflags "-X github.com/aoxn/ovm.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built ovm binary to build/bin/ovm.arm64
	rm -f build/bin/ovm.arm64
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/ovm.arm64 build/bin/ovm.arm64.unzip
	rm -f build/bin/ovm.arm64.unzip

log:
	kubectl --kubeconfig ~/.kube/config.ovm -n kube-system scale deploy ovm --replicas 0
	kubectl --kubeconfig ~/.kube/config.ovm -n kube-system scale deploy ovm --replicas 1
	@echo "wait 8 seconds..." ; sleep 8
	kubectl --kubeconfig ~/.kube/config.ovm -n kube-system logs -l app=ovm -f

# standard cleanup target
clean: clean-cache clean-output

deploy: ovmlinux image push

build: ovmmac
	build/bin/ovm build --arch amd64 --os centos --ovm-version 0.1.1 --run-version 2.0
	cp -rf build/bin/ovm /usr/local/bin/ovm

release-mac:
	build/bin/ovm build --arch darwin --os macos --ovm-version 0.1.1

build-all: ovmmac
	build/bin/ovm build \
			--arch amd64 \
			--os centos \
			--ovm-version 0.1.1 \
			--run-version 2.0 \
			--kubernetes-version 1.20.4-aliyun.1 \
			--kubernetes-cni-version 0.8.6 \
			--etcd-version v3.4.3 \
			--runtime-version 19.03.5
	cp -rf build/bin/ovm /usr/local/bin/ovm

release: build deploy

.PHONY: all make-cache clean-cache out-dir clean-output ovm build install clean
