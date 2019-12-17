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
all: test build

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

.PHONY: docker
docker:
	docker build -t $(IMAGE_NAME) .
	@echo built image $(IMAGE_NAME)
