GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

PROFILE ?= integrations
REGIONS ?= af-south-1 ap-east-1 ap-northeast-1 ap-northeast-2 ap-south-1 ap-southeast-1 ap-southeast-2 ca-central-1 eu-central-1 eu-north-1 eu-south-1 eu-west-1 eu-west-2 eu-west-3 me-south-1 sa-east-1 us-east-1 us-east-2 us-west-1 us-west-2
LAYER_NAME ?= signalfx-extension-wrapper

VERSION=`git log --format=format:%h -1`

all: clean build package

clean:
	$(GOCLEAN)
	rm -rf bin

build:
	mkdir -p bin/extensions
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/extensions -ldflags "-s -w -X main.gitVersion=$(VERSION)" ./cmd/signalfx-extension-wrapper/*.go

package:
	cd bin; zip -r extensions.zip * -x '**/*.zip'

add-layer-version:
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	ZIP_NAME="$(PWD)/bin/extensions.zip" \
	REGIONS="$(REGIONS)" \
		scripts/add-layer-version.sh

add-layer-version-permission:
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(REGIONS)" \
		scripts/add-layer-version-permission.sh

list-layer-versions:
	PROFILE="$(PROFILE)" \
	LAYER_NAME="$(LAYER_NAME)" \
	REGIONS="$(REGIONS)" \
		scripts/list-layer-versions.sh
