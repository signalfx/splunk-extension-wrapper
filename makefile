GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

BIN_DIR := bin
REGIONS_PATH := $(BIN_DIR)/regions
EXTENSIONS_DIR := $(BIN_DIR)/extensions

PROFILE ?= integrations
LAYER_NAME ?= signalfx-extension-wrapper

VERSION=`git log --format=format:%h -1`

ci_check = $(if $(CI_BUILD),,$(error you're not allowed to do that - set a CI_BUILD variable if you know what you're doing))

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
	AWS_PROFILE="$(PROFILE)" aws ec2 describe-regions | jq -r '.Regions | map(.RegionName) | join(" ")' > $(REGIONS_PATH)

.PHONY: add-layer-version
add-layer-version: $(REGIONS_PATH)
	$(call ci_check)
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	ZIP_NAME="$(PWD)/$(BIN_DIR)/extensions.zip" \
	REGIONS="$(shell cat $(REGIONS_PATH))" \
		scripts/add-layer-version.sh

.PHONY: add-layer-version-permission
add-layer-version-permission: $(REGIONS_PATH)
	$(call ci_check)
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(shell cat $(REGIONS_PATH))" \
		scripts/add-layer-version-permission.sh

.PHONY: list-layer-versions
list-layer-versions: $(REGIONS_PATH)
	$(call ci_check)
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(shell cat $(REGIONS_PATH))" \
		scripts/list-layer-versions.sh
