GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

BIN_DIR := bin
REGIONS_PATH := $(BIN_DIR)/regions
EXTENSIONS_DIR := $(BIN_DIR)/extensions

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
	cd $(BIN_DIR); zip -r extensions.zip * -x '**/*.zip'


$(REGIONS_PATH):
	mkdir -p $(BIN_DIR)
ifndef REGIONS
	AWS_PROFILE="$(PROFILE)" aws ec2 describe-regions --query "Regions[].RegionName" --output text > $(REGIONS_PATH)
else
	echo $(REGIONS) > $(REGIONS_PATH)
endif

.PHONY: supported-regions
supported-regions: $(REGIONS_PATH)
	$(eval REGIONS ?= $(shell cat $(REGIONS_PATH)))

.PHONY: ci-check
ci-check:
	$(if $(CI),,$(error you're not allowed to do that - set a CI variable if you know what you're doing))


.PHONY: add-layer-version
add-layer-version: ci-check supported-regions
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	ZIP_NAME="$(PWD)/$(BIN_DIR)/extensions.zip" \
	REGIONS="$(REGIONS)" \
		scripts/add-layer-version.sh

.PHONY: add-layer-version-permission
add-layer-version-permission: ci-check supported-regions
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(REGIONS)" \
		scripts/add-layer-version-permission.sh

.PHONY: list-layer-versions
list-layer-versions: supported-regions
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(REGIONS)" \
		scripts/list-layer-versions.sh
