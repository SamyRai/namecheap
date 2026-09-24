# ZoneKit Makefile

.PHONY: build test clean install lint fmt vet deps help

# Variables
BINARY_NAME=zonekit
MAIN_PATH=./main.go
BUILD_DIR=build
VERSION_PKG=go.glpx.pro/zonekit/pkg/version

# VERSION defaults to the nearest git tag (falling back to "dev" outside a
# git checkout, e.g. an extracted release tarball); override on the command
# line for a specific release build: `make build VERSION=1.2.3`. Injected via
# -ldflags so pkg/version.Version never needs a hardcoded, driftable literal
# (O7 — see pkg/version/version.go).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -w -s \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).GitCommit=$(COMMIT) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE)
GOFLAGS=-ldflags="$(LDFLAGS)"

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-all: ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PATH)

# Testing and Quality
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	@echo "Running golangci-lint v2..."
	@golangci-lint --version
	@golangci-lint run

lint-fix: ## Run linter with auto-fix
	@echo "Running golangci-lint v2 with auto-fix..."
	@golangci-lint run --fix

fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

deps-vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	go mod vendor

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Development helpers
run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	@echo "Installing golangci-lint v2..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.6.2
	@golangci-lint --version

# Configuration helpers
config-example: ## Show example configuration
	@echo "Example configuration:"
	@cat configs/config.example.yaml

# Version
version: ## Show the version that `make build` would inject
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"

# Releases are cut by pushing a git tag (`git tag vX.Y.Z && git push --tags`),
# which .github/workflows/release.yml builds and injects via -ldflags above.
# There is no source-code version literal to bump.

# Release
release-check: lint test ## Check if ready for release
	@echo "Release checks passed!"

.DEFAULT_GOAL := help
