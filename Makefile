VERSION ?= $(shell git describe --tags --dirty --always --match=v* || echo v0)
BUILD := $(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X=main.version=$(VERSION) -X=main.build=$(BUILD)"
BUILDFLAGS=$(LDFLAGS)
PROJECTNAME=telnet
GOEXE := $(shell go env GOEXE)
GOPATH := $(shell go env GOPATH)
GOOS := $(shell go env GOOS)
BIN=bin/$(PROJECTNAME)$(GOEXE)
IMPORT_PATH := /usr/local/include
IMPORT_PATH_WIN := c:\protobuf\include

ifneq ($(GOOS), windows)
	RACE = -race
	PWD := $(shell pwd)
endif

ifeq ($(GOOS), windows)
	IMPORT_PATH := $(IMPORT_PATH_WIN)
	PWD := $(shell echo %cd%)
endif

export

.PHONY: setup
setup: ## Install all the build and lint dependencies
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install github.com/golang/protobuf/protoc-gen-go
	go get ./...

.PHONY: test
test: ## Run all the tests
	go test -cover $(RACE) -v $(BUILDFLAGS) ./...

.PHONY: lint
lint: ## Run all the linters
	golangci-lint run --enable-all --disable gochecknoinits --disable gochecknoglobals --disable goimports \
	--out-format=tab --tests=false ./...

.PHONY: ci
ci: setup lint test build ## Run all the tests and code checks

.PHONY: generate
generate:
	go generate -v ./...

.PHONY: build
build: mod-refresh ## Build a version
	go build $(BUILDFLAGS) -o $(BIN)

.PHONY: install
install: mod-refresh ## Install a binary
	go install $(BUILDFLAGS)

.PHONY: clean
clean: ## Remove temporary files
	go clean

.PHONY: mod-refresh
mod-refresh: clean ## Refresh modules
	go mod tidy -v

.PHONY: version
version:
	@echo $(VERSION)-$(BUILD)

.PHONY: release
release:
	git tag $(ver)
	git push origin --tags

.DEFAULT_GOAL := build
