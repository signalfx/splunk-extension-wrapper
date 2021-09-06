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

SHELL := /bin/bash

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

BIN_DIR := $(PWD)/bin
EXTENSIONS_DIR := $(BIN_DIR)/extensions
EXTENSION_ZIP := $(BIN_DIR)/extension.zip

VERSION=`git log --format=format:%h -1`

all: clean license-check build package

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

.PHONY: build
build:
	mkdir -p $(EXTENSIONS_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(EXTENSIONS_DIR) -ldflags "-s -w -X main.gitVersion=$(VERSION)" ./cmd/signalfx-extension-wrapper/*.go

.PHONY: package
package:
	cd $(BIN_DIR); zip -r $(EXTENSION_ZIP) $(shell realpath --relative-to $(BIN_DIR) $(EXTENSIONS_DIR))/*

.PHONY: license-check
license-check:
	@licRes=$$(for f in $$(find . -type f \( -iname '*.go' -o -iname '*.sh' -o -iname '*.js' \) ! -path '**/third_party/*') ; do \
	           awk '/Copyright Splunk Inc|generated|GENERATED/ && NR<=3 { found=1; next } END { if (!found) print FILENAME }' $$f; \
	   done); \
	   if [ -n "$${licRes}" ]; then \
	           echo "license header checking failed:"; echo "$${licRes}"; \
	           exit 1; \
	   fi