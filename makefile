SHELL := /bin/bash

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

BIN_DIR := $(PWD)/bin
EXTENSIONS_DIR := $(BIN_DIR)/extensions
EXTENSION_ZIP := $(BIN_DIR)/extension.zip

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
	cd $(BIN_DIR); zip -r $(EXTENSION_ZIP) $(shell realpath --relative-to $(BIN_DIR) $(EXTENSIONS_DIR))/*