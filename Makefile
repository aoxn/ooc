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

# Simple makefile to build wdrip quickly and reproducibly in a container
# Only requires docker on the host

OS_TYPE :=$(shell echo `uname`|tr '[A-Z]' '[a-z]')

# settings
REPO_ROOT:=${CURDIR}
# autodetect host GOOS and GOARCH by default, even if go is not installed
GOOS?=$(shell hack/util/goos.sh)
GOARCH?=$(shell hack/util/goarch.sh)
REGISTRY=registry.cn-hangzhou.aliyuncs.com/aoxn/wdrip
TAG?=$(shell hack/util/tag.sh)

# make install will place binaries here
# the default path attempst to mimic go install
INSTALL_DIR?=$(shell hack/util/goinstalldir.sh)

# the output binary name, overridden when cross compiling
KIND_BINARY_NAME?=wdrip
# use the official module proxy by default
GOPROXY?=https://mirrors.aliyun.com/goproxy
# default build image
GO_VERSION?=1.14.3
GO_IMAGE?=golang:$(GO_VERSION)
# docker volume name, used as a go module / build cache
CACHE_VOLUME?=wdrip-build-cache

# variables for consistent logic, don't override these
CONTAINER_REPO_DIR=/src/wdrip
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

.PHONY: codesign
codesign:
	@echo "codesign begin"
	codesign --entitlements wdrip.entitlements -s - ./build/bin/wdrip || true

# builds wdrip in a container, outputs to $(OUT_DIR)
wdrip: make-cache out-dir
	@echo + Building wdrip binary
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
		    -ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built wdrip binary to $(OUT_DIR)/$(KIND_BINARY_NAME).unzip
	rm -f build/bin/wdrip
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/wdrip $(OUT_DIR)/$(KIND_BINARY_NAME).unzip
	rm -f $(OUT_DIR)/$(KIND_BINARY_NAME).unzip

# alias for building wdrip
buildwdrip: wdrip


image:
	docker build -t $(REGISTRY):$(TAG) build/

bimage:
	docker build -t $(REGISTRY):$(TAG) build/

push: bimage
	docker push $(REGISTRY):$(TAG)

# use: make install INSTALL_DIR=/usr/local/bin
install: build
	@echo + Copying wdrip binary to INSTALL_DIR
	install $(OUT_DIR)/$(KIND_BINARY_NAME) $(INSTALL_DIR)/$(KIND_BINARY_NAME)


#-X gitlab.alibaba-inc.com/cos/ros.Template=$(ROS_TPL)
wdripmac:
	GOARCH=amd64 \
	GOOS=darwin \
	CGO_ENABLED=1 \
	GO111MODULE=on \
	go build -v -o build/bin/wdrip.unzip \
	-ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built wdrip binary to build/bin/wdrip
	rm -f build/bin/wdrip
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/wdrip build/bin/wdrip.unzip
	rm -f build/bin/wdrip.unzip

omac:
	@echo + Built wdrip binary to /usr/local/bin/wdrip
	GOARCH=amd64 \
	GOOS=darwin \
	CGO_ENABLED=1 \
	GO111MODULE=on \
	go build -v -o build/bin/wdrip.unzip \
	-ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
	mv build/bin/wdrip.unzip /usr/local/bin/wdrip
	codesign --entitlements wdrip.entitlements -s - /usr/local/bin/wdrip || true

olinux:
	@echo + Built wdrip binary to /usr/local/bin/wdrip
	GOARCH=amd64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/wdrip.unzip \
	-ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
	mv build/bin/wdrip.unzip /usr/local/bin/wdrip.amd64
	codesign --entitlements wdrip.entitlements -s - /usr/local/bin/wdrip.amd64 || true

wdriplinux:
	GOARCH=amd64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/wdrip.amd64.unzip \
	-ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built wdrip binary to build/bin/wdrip.amd64
	rm -f build/bin/wdrip.amd64
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/wdrip.amd64 build/bin/wdrip.amd64.unzip
	rm -f build/bin/wdrip.amd64.unzip

wdripwin:
	GOARCH=amd64 \
	GOOS=windows \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/wdrip.exe \
       -ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
wdriparm64:
	GOARCH=arm64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/wdrip.arm64.unzip \
	-ldflags "-X github.com/aoxn/wdrip.Version=$(TAG) -s -w" cmd/main.go
	@echo + Built wdrip binary to build/bin/wdrip.arm64
	rm -f build/bin/wdrip.arm64
	build/tool/upx.$(OS_TYPE) -9 -o build/bin/wdrip.arm64 build/bin/wdrip.arm64.unzip
	rm -f build/bin/wdrip.arm64.unzip

log:
	kubectl --kubeconfig ~/.kube/config.wdrip -n kube-system scale deploy wdrip --replicas 0
	kubectl --kubeconfig ~/.kube/config.wdrip -n kube-system scale deploy wdrip --replicas 1
	@echo "wait 8 seconds..." ; sleep 8
	kubectl --kubeconfig ~/.kube/config.wdrip -n kube-system logs -l app=wdrip -f

# standard cleanup target
clean: clean-cache clean-output

deploy: wdriplinux image push

build: wdripmac
	build/bin/wdrip build --arch amd64 --os centos --wdrip-version 0.1.1 --run-version 2.0
	cp -rf build/bin/wdrip /usr/local/bin/wdrip

release-mac:
	build/bin/wdrip build --arch darwin --os macos --wdrip-version 0.1.1

build-all: wdripmac
	build/bin/wdrip build \
			--arch amd64 \
			--os centos \
			--wdrip-version 0.1.1 \
			--run-version 2.0 \
			--kubernetes-version 1.20.4-aliyun.1 \
			--kubernetes-cni-version 0.8.6 \
			--etcd-version v3.4.3 \
			--runtime-version 19.03.5
	cp -rf build/bin/wdrip /usr/local/bin/wdrip

release: build deploy

.PHONY: all make-cache clean-cache out-dir clean-output wdrip build install clean
