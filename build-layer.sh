#!/bin/bash

# Copyright Splunk Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# install dependencies
echo "Installing dependencies"
[[ -d /go/pkg/mod/cache ]] && exit 0
# retry up to 3 times in case of network issues
for i in $(seq 1 3); do
    go mod download && break
    sleep 10
done

# run local tests
echo "Running local tests"

mkdir ~/testresults
(cd /tmp || exit; GO111MODULE=on go get gotest.tools/gotestsum)
CGO_ENABLED=0 gotestsum --format short-verbose --junitfile ~/testresults/unit.xml --raw-command -- go test --json -p 4 ./...

# package the layer
echo "Packaging the layer"
apt update && apt install --assume-yes zip
make