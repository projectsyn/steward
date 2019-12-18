# Antora variables
pages   := $(shell find . -type f -name '*.adoc')
web_dir := ./_antora

docker_cmd  ?= docker
docker_opts ?= --rm --tty --user "$$(id -u)"

antora_cmd  ?= $(docker_cmd) run $(docker_opts) --volume "$${PWD}":/antora vshn/antora:1.3
antora_opts ?= --cache-dir=.cache/antora

vale_cmd ?= $(docker_cmd) run $(docker_opts) --volume "$${PWD}"/docs/modules/ROOT/pages:/pages vshn/vale:1.1 --minAlertLevel=error /pages


# Project parameters
BINARY_NAME=steward

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VERSION=$(shell git describe --tags --always --dirty --match=v* || (echo "command failed $$?"; exit 1))
IMAGE_NAME=docker.io/vshn/$(BINARY_NAME):$(VERSION)

.PHONY: all
all: test build docs

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -v \
		-o $(BINARY_NAME) \
		-ldflags "-X main.Version=$(VERSION)"
	@echo built '$(VERSION)'

.PHONY: test
test:
	$(GOTEST) -v ./...

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(web_dir)

.PHONY: docker
docker:
	docker build -t $(IMAGE_NAME) .
	@echo built image $(IMAGE_NAME)

.PHONY: docs
docs:    $(web_dir)/index.html

$(web_dir)/index.html: playbook.yml $(pages)
	$(antora_cmd) $(antora_opts) $<

.PHONY: check
check:
	$(vale_cmd)


