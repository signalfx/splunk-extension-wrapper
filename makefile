GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

PROFILE ?= integrations
REGION ?= us-east-1
NAME ?= lambda-extension-wrapper

all: clean build package

clean:
	$(GOCLEAN)
	rm -rf bin

build:
	mkdir -p bin/extensions
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/extensions -ldflags "-s -w" ./cmd/lambda-extension/*.go


package:
	# this "magic" file enables access to Lambda-managed runtimes
	# this is only required during preview access
	touch bin/preview-extensions-ggqizro707
	cd bin; zip -r extensions.zip * -x '**/*.zip'

publish:
	aws --profile=$(PROFILE) --region=$(REGION) lambda publish-layer-version --layer-name $(NAME) --zip-file 'fileb://bin/extensions.zip'
