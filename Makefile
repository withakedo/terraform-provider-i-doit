BINARY   := terraform-provider-i-doit
VERSION  ?= 0.1.0
OS_ARCH  ?= $(shell go env GOOS)_$(shell go env GOARCH)
PLUGIN_DIR := $(HOME)/.terraform.d/plugins/registry.terraform.io/withakedo/i-doit/$(VERSION)/$(OS_ARCH)

GOFLAGS ?=

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build           Compile the provider binary"
	@echo "  install         Build and install into ~/.terraform.d/plugins for local testing"
	@echo "  tidy            go mod tidy"
	@echo "  fmt             gofmt -s -w"
	@echo "  vet             go vet ./..."
	@echo "  test            Unit tests"
	@echo "  testacc         Acceptance tests (needs TF_ACC=1 and IDOIT_URL / IDOIT_APIKEY)"
	@echo "  docs            Regenerate docs/ with tfplugindocs"
	@echo "  release         Local, manual GoReleaser build (no publishing)"

.PHONY: build
build:
	go build $(GOFLAGS) -o $(BINARY) .

.PHONY: install
install:
	mkdir -p "$(PLUGIN_DIR)"
	go build $(GOFLAGS) -o "$(PLUGIN_DIR)/$(BINARY)_v$(VERSION)" .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test $(GOFLAGS) ./... -count=1

.PHONY: testacc
testacc:
	TF_ACC=1 go test $(GOFLAGS) ./internal/provider/... -v -count=1 -timeout 30m

.PHONY: docs
docs:
	go generate ./...

.PHONY: release
release:
	goreleaser release --clean --snapshot
