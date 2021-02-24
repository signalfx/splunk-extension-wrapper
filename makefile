SHELL := /bin/bash

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

BIN_DIR := $(PWD)/bin
REGIONS_PATH := $(BIN_DIR)/regions
EXTENSIONS_DIR := $(BIN_DIR)/extensions
EXTENSTION_ZIP := $(BIN_DIR)/extension.zip
VERSIONS_FILE := $(BIN_DIR)/versions

TEST_DIR := $(BIN_DIR)/test
FUNCTION_PATH := $(TEST_DIR)/function.zip
FUNCTION_NAME := singalfx-extension-wrapper-test-function

PROFILE ?= integrations
LAYER_NAME ?= signalfx-extension-wrapper

VERSION=`git log --format=format:%h -1`

all: clean build package

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
	cd $(BIN_DIR); zip -r $(EXTENSTION_ZIP) $(shell realpath --relative-to $(BIN_DIR) $(EXTENSIONS_DIR))/*


$(REGIONS_PATH):
	mkdir -p $(BIN_DIR)
ifndef REGIONS
	AWS_PROFILE="$(PROFILE)" aws ec2 describe-regions --query "Regions[].RegionName" --output text > $(REGIONS_PATH)
else
	echo $(REGIONS) > $(REGIONS_PATH)
endif

.PHONY: supported-regions
supported-regions: $(REGIONS_PATH)
	$(eval REGIONS = $(shell cat $(REGIONS_PATH)))

.PHONY: ci-check
ci-check:
	$(if $(CI),,$(error you're not allowed to do that - set a CI variable if you know what you're doing))


.PHONY: add-layer-version
add-layer-version: ci-check supported-regions
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	ZIP_NAME="$(EXTENSTION_ZIP)" \
	REGIONS="$(REGIONS)" \
	VERSIONS_FILE="$(VERSIONS_FILE)" \
		scripts/add-layer-version.sh

.PHONY: add-layer-version-permission
add-layer-version-permission: ci-check supported-regions
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(REGIONS)" \
	VERSIONS_FILE="$(VERSIONS_FILE)" \
		scripts/add-layer-version-permission.sh

.PHONY: list-latest-versions
list-latest-versions: supported-regions
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(REGIONS)" \
		scripts/list-layer-versions.sh


$(FUNCTION_PATH): test/function/index.js
	mkdir -p $(TEST_DIR)

	cd test/function; zip -r $(FUNCTION_PATH) index.js

$(TEST_DIR)/%.json: test/%.json.template $(FUNCTION_PATH)
	mkdir -p $(TEST_DIR)
	cat $< | \
		FUNCTION_ZIP="$(shell base64 -i $(FUNCTION_PATH))" \
		FUNCTION_LAYER="$(shell grep $(REGION) $(VERSIONS_FILE))" \
		FUNCTION_NAME="$(FUNCTION_NAME)" \
		FUNCTION_INGEST="$(FUNCTION_INGEST)" \
		FUNCTION_TOKEN="$(FUNCTION_TOKEN)" \
		envsubst > $@

.PHONY: run-test
run-test: $(TEST_DIR)/add-test-function.json $(TEST_DIR)/delete-test-function.json
	PROFILE="$(PROFILE)" \
	REGION="$(REGION)" \
	FUNCTION_NAME="$(FUNCTION_NAME)" \
		scripts/run-test-in-region.sh

.PHONY: verify-test
verify-test:
	cd test/verify; npm i; FUNCTION_NAME="$(FUNCTION_NAME)" node invocations_watcher.js
