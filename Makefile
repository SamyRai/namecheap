# ZoneKit Makefile

.PHONY: build test clean install lint fmt vet deps help

# Variables
BINARY_NAME=zonekit
MAIN_PATH=./main.go
BUILD_DIR=build
GOFLAGS=-ldflags="-w -s"

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

test-conformance: ## Run conformance tests
	@echo "Running conformance tests..."
	go test -v ./pkg/dns/provider/... -run Conformance

lint: ## Run linter
	@echo "Running golangci-lint v2..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	elif [ -f $$(go env GOPATH)/bin/golangci-lint ]; then \
		$$(go env GOPATH)/bin/golangci-lint run; \
	else \
		echo "golangci-lint not found"; \
		exit 1; \
	fi

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
version: ## Show current version
	@echo "Current version: $$(grep 'Version = ' pkg/version/version.go | sed 's/.*Version = "\(.*\)"/\1/')"

version-bump-patch: ## Bump patch version (0.1.0 -> 0.1.1)
	@./scripts/bump-version.sh patch

version-bump-minor: ## Bump minor version (0.1.0 -> 0.2.0)
	@./scripts/bump-version.sh minor

version-bump-major: ## Bump major version (0.1.0 -> 1.0.0)
	@./scripts/bump-version.sh major

# Release
release-check: lint test ## Check if ready for release
	@echo "Release checks passed!"

.DEFAULT_GOAL := help
