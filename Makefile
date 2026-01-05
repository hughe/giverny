.PHONY: all build clean test test-binary integration-test run install image

all: build

# Binary name
BINARY_NAME=giverny
BUILD_DIR=bin

# Version information
VERSION_TAG=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION_BRANCH=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Test environment directory - defaults to unique temp dir if not already set
# Use ?= to allow override via environment variable or command line
# $$$$ expands to process ID for uniqueness
GIV_TEST_ENV_DIR?=/tmp/giverny-test-env-$$$$

# Build the project
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.versionTag=$(VERSION_TAG) -X main.versionHash=$(VERSION_HASH) -X main.versionBranch=$(VERSION_BRANCH)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/giverny

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Run tests with environment setup and teardown
# Pass additional arguments via GO_TEST_ARGS env var
test:
	@echo "Setting up test environment..."
	@export GIV_TEST_ENV_DIR=$(GIV_TEST_ENV_DIR) && \
	./scripts/setup-test-env.sh && \
	(GIV_TEST_ENV_DIR=$$GIV_TEST_ENV_DIR go test -v $(GO_TEST_ARGS) ./...; TEST_RESULT=$$?; \
	./scripts/teardown-test-env.sh; \
	exit $$TEST_RESULT)

# Test the giverny binary
test-binary: build
	@echo "Testing giverny binary..."
	@export GIV_TEST_ENV_DIR=$(GIV_TEST_ENV_DIR) && \
	./scripts/setup-test-env.sh && \
	(cd $$GIV_TEST_ENV_DIR && \
	$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) --help > /dev/null && \
	echo "Binary test: help command OK"; \
	TEST_RESULT=$$?; \
	cd $(CURDIR) && \
	./scripts/teardown-test-env.sh; \
	exit $$TEST_RESULT)

# Run integration tests with environment setup and teardown
# Pass additional arguments via GO_TEST_ARGS env var
integration-test:
	@echo "Setting up test environment..."
	@export GIV_TEST_ENV_DIR=$(GIV_TEST_ENV_DIR) && \
	./scripts/setup-test-env.sh && \
	(INTEGRATION_TEST=1 GIV_TEST_ENV_DIR=$$GIV_TEST_ENV_DIR go test -v $(GO_TEST_ARGS) ./...; TEST_RESULT=$$?; \
	./scripts/teardown-test-env.sh; \
	exit $$TEST_RESULT)

# Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install ./cmd/giverny

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

image:
	@echo "Building Docker image for Giverny to work on Giverny ..."
	cd docker && docker build -t giverny-builder . 

# Show help
help:
	@echo "Available targets:"
	@echo "  build            - Build the binary"
	@echo "  clean            - Remove build artifacts"
	@echo "  test             - Run tests with environment setup/teardown"
	@echo "  test-binary      - Test the giverny binary"
	@echo "  integration-test - Run integration tests with INTEGRATION_TEST=1"
	@echo "  install          - Install to GOPATH/bin"
	@echo "  fmt              - Format code"
	@echo "  lint             - Run linter"
	@echo "  help             - Show this help message"
