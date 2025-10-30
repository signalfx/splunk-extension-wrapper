#!/bin/bash -e

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
    set +e
    go mod download && break
    set -e
    sleep 10
done

# run local tests
echo "Running local tests"

mkdir -p ~/testresults
go install gotest.tools/gotestsum@latest

# Ensure Go bin directory is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"

CGO_ENABLED=0 gotestsum --format short-verbose --junitfile ~/testresults/unit.xml --raw-command -- go test --json -p 4 ./...

# package the layer
echo "Packaging the layer"
# Install zip if needed (apt for Linux, brew for macOS)
if command -v apt &> /dev/null; then
    apt update && apt install --assume-yes zip
elif command -v brew &> /dev/null; then
    # zip is typically pre-installed on macOS, but install if missing
    command -v zip &> /dev/null || brew install zip
fi
make
