#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
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

set -x
set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")
# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
# Alert: `go mod vendor` before running this script in go.mod
echo "Alert: run 'go mod vendor' before running this script in go.mod. And delete vendor dir before 'make wdripmac'"

CACHE="~/.wdrip"
mkdir -p "$CACHE"
CODEGEN_PKG="$CACHE/code-generator"

if [[ ! -d "$CODEGEN_PKG" ]];
then
    pushd "$CACHE";git clone https://github.com/kubernetes/code-generator.git; popd
fi

bash "${CODEGEN_PKG}"/generate-groups.sh all \
  github.com/aoxn/wdrip/pkg/generated github.com/aoxn/wdrip/pkg/apis \
  alibabacloud.com:v1 \
  --output-base "${SCRIPT_ROOT}/../.." \
  --go-header-file "${SCRIPT_ROOT}"/boilerplate.go.txt

# To use your own boilerplate text append:
#   --go-header-file "${SCRIPT_ROOT}"/hack/custom-boilerplate.go.txt

# hack
PROJECT=${SCRIPT_ROOT}/../..
DEEPCOPY_LOC="$PROJECT"/github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1/zz_generated.deepcopy.go
mv "$DEEPCOPY_LOC" "${PROJECT}"/pkg/apis/alibabacloud.com/v1
rm -rf "${PROJECT}"/pkg/generated
mv "$PROJECT"/github.com/aoxn/wdrip/pkg/generated "${PROJECT}"/pkg/
rm -rf github.com
