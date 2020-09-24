GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

PROFILE ?= integrations
REGION ?= us-east-1
NAME ?= signalfx-extension-wrapper

VERSION=`git log --format=format:%h -1`

all: clean build package

clean:
	$(GOCLEAN)
	rm -rf bin

build:
	mkdir -p bin/extensions
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/extensions -ldflags "-s -w -X main.gitVersion=$(VERSION)" ./cmd/signalfx-extension-wrapper/*.go

package:
	# this "magic" file enables access to Lambda-managed runtimes
	# this is only required during preview access
	touch bin/preview-extensions-ggqizro707
	cd bin; zip -r extensions.zip * -x '**/*.zip'

publish:
	aws --profile=$(PROFILE) --region=$(REGION) lambda publish-layer-version --layer-name $(NAME) --zip-file 'fileb://bin/extensions.zip'
