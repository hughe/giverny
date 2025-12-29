.PHONY: build clean test test-with-env test-binary run install

# Binary name
BINARY_NAME=giverny
BUILD_DIR=build
TEST_ENV_DIR?=/tmp/giverny-test-env-$$$$

# Build the project
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/giverny

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with environment setup and teardown
test-with-env:
	@echo "Setting up test environment..."
	@export TEST_ENV_DIR=$(TEST_ENV_DIR) && \
	./scripts/setup-test-env.sh && \
	(go test -v ./...; TEST_RESULT=$$?; \
	./scripts/teardown-test-env.sh; \
	exit $$TEST_RESULT)

# Test the giverny binary
test-binary: build
	@echo "Testing giverny binary..."
	@export TEST_ENV_DIR=$(TEST_ENV_DIR) && \
	./scripts/setup-test-env.sh && \
	(cd $$TEST_ENV_DIR && \
	$(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) --help > /dev/null && \
	echo "Binary test: help command OK"; \
	TEST_RESULT=$$?; \
	cd $(CURDIR) && \
	./scripts/teardown-test-env.sh; \
	exit $$TEST_RESULT)

# Run the application
run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

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

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-with-env  - Run tests with environment setup/teardown"
	@echo "  test-binary    - Test the giverny binary"
	@echo "  run            - Build and run the application"
	@echo "  install        - Install to GOPATH/bin"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
	@echo "  help           - Show this help message"
