# Project parameters
BINARY_NAME ?= steward

VERSION ?= $(shell git describe --tags --always --dirty --match=v* || (echo "command failed $$?"; exit 1))

IMAGE_NAME ?= docker.io/vshn/$(BINARY_NAME):$(VERSION)

docker_cmd  ?= docker
docker_opts ?= --rm --tty --user "$$(id -u)"

vale_cmd           ?= $(docker_cmd) run $(docker_opts) --volume "$${PWD}"/docs/modules/ROOT/pages:/pages vshn/vale:2.1.1 --minAlertLevel=error --config=/pages/.vale.ini /pages
antora_preview_cmd ?= $(docker_cmd) run --rm --publish 2020:2020 --volume "${PWD}":/antora vshn/antora-preview:2.3.3 --style=syn --antora=docs

# Go parameters
GOCMD   ?= go
GOBUILD ?= $(GOCMD) build
GOCLEAN ?= $(GOCMD) clean
GOTEST  ?= $(GOCMD) test
GOGET   ?= $(GOCMD) get

.PHONY: all
all: test build docs

.PHONY: generate
generate:
	go generate main.go

.PHONY: build
build: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -v \
		-o $(BINARY_NAME) \
		-ldflags "-X main.Version=$(VERSION)"
	@echo built '$(VERSION)'

.PHONY: test
test: generate
	$(GOTEST) -v ./...

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

.PHONY: docker
docker:
	docker build -t $(IMAGE_NAME) .
	@echo built image $(IMAGE_NAME)

.PHONY: docs-serve
docs-serve:
	$(antora_preview_cmd)

.PHONY: docs-vale
docs-vale:
	$(vale_cmd)
