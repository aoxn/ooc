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

# Simple makefile to build ooc quickly and reproducibly in a container
# Only requires docker on the host

# settings
REPO_ROOT:=${CURDIR}
# autodetect host GOOS and GOARCH by default, even if go is not installed
GOOS?=$(shell hack/util/goos.sh)
GOARCH?=$(shell hack/util/goarch.sh)
REGISTRY=registry.cn-hangzhou.aliyuncs.com/aoxn/ooc
TAG?=$(shell hack/util/tag.sh)

# make install will place binaries here
# the default path attempst to mimic go install
INSTALL_DIR?=$(shell hack/util/goinstalldir.sh)

# the output binary name, overridden when cross compiling
KIND_BINARY_NAME?=ooc
# use the official module proxy by default
GOPROXY?=https://mirrors.aliyun.com/goproxy
# default build image
GO_VERSION?=1.14.3
GO_IMAGE?=golang:$(GO_VERSION)
# docker volume name, used as a go module / build cache
CACHE_VOLUME?=ooc-build-cache

# variables for consistent logic, don't override these
CONTAINER_REPO_DIR=/src/ooc
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

# builds ooc in a container, outputs to $(OUT_DIR)
ooc: make-cache out-dir
	@echo + Building ooc binary
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
		go build -v -o /out/$(KIND_BINARY_NAME) \
		    -ldflags "-X github.com/aoxn/ooc.Version=$(TAG) -s -w" .
	@echo + Built ooc binary to $(OUT_DIR)/$(KIND_BINARY_NAME)

# alias for building ooc
buildooc: ooc


image:
	docker build -t $(REGISTRY):$(TAG) build/

bimage:
	docker build -t $(REGISTRY):$(TAG) build/

push: bimage
	docker push $(REGISTRY):$(TAG)

# use: make install INSTALL_DIR=/usr/local/bin
install: build
	@echo + Copying ooc binary to INSTALL_DIR
	install $(OUT_DIR)/$(KIND_BINARY_NAME) $(INSTALL_DIR)/$(KIND_BINARY_NAME)


#-X gitlab.alibaba-inc.com/cos/ros.Template=$(ROS_TPL)
oocmac:
	GOARCH=amd64 \
	GOOS=darwin \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ooc \
       -ldflags "-X github.com/aoxn/ooc.Version=$(TAG) -s -w" cmd/main.go

ooclinux:
	GOARCH=amd64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ooc.amd64 \
       -ldflags "-X github.com/aoxn/ooc.Version=$(TAG) -s -w" cmd/main.go
oocwin:
	GOARCH=amd64 \
	GOOS=windows \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ooc.exe \
       -ldflags "-X github.com/aoxn/ooc.Version=$(TAG) -s -w" cmd/main.go
oocarm64:
	GOARCH=arm64 \
	GOOS=linux \
	CGO_ENABLED=0 \
	GO111MODULE=on \
	go build -v -o build/bin/ooc.arm64 \
       -ldflags "-X github.com/aoxn/ooc.Version=$(TAG) -s -w" cmd/main.go

log:
	kubectl --kubeconfig ~/.kube/config.ooc -n kube-system scale deploy ooc --replicas 0
	kubectl --kubeconfig ~/.kube/config.ooc -n kube-system scale deploy ooc --replicas 1
	@echo "wait 8 seconds..." ; sleep 8
	kubectl --kubeconfig ~/.kube/config.ooc -n kube-system logs -l app=ooc -f

# standard cleanup target
clean: clean-cache clean-output

deploy: ooclinux image push

build: oocmac
	build/bin/ooc build --arch amd64 --os centos --ooc-version 0.1.1 --run-version 2.0

release: build deploy

.PHONY: all make-cache clean-cache out-dir clean-output ooc build install clean
