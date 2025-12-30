.PHONY: build test lint clean install fmt vet tidy help

# Variables
BINARY_NAME=nomos-provider-terraform-remote-state
BUILD_DIR=bin
GO=go
GOLANGCI_LINT=golangci-lint
COVERAGE_FILE=coverage.out

# Version information
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

## help: Display this help message
help:
	@echo "Nomos Terraform Remote State Provider - Build Targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## build: Build the provider binary
build: tidy
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## install: Install the provider binary to GOPATH/bin
install: tidy
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) ./cmd/$(BINARY_NAME)
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "Coverage: $(shell $(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}')"

## test-coverage: Run tests and generate HTML coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linters
lint:
	@echo "Running linters..."
	@which $(GOLANGCI_LINT) > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	$(GOLANGCI_LINT) run --timeout=5m ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	$(GO) mod tidy

## clean: Remove build artifacts and cache
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html
	$(GO) clean -cache -testcache
	@echo "Clean complete"

## run: Build and run the provider
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

## verify: Run all verification steps (fmt, vet, lint, test)
verify: fmt vet lint test
	@echo "Verification complete"
