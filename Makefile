.PHONY: build clean test run install

# Binary name
BINARY_NAME=giverny
BUILD_DIR=build

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
	@echo "  build   - Build the binary"
	@echo "  clean   - Remove build artifacts"
	@echo "  test    - Run tests"
	@echo "  run     - Build and run the application"
	@echo "  install - Install to GOPATH/bin"
	@echo "  fmt     - Format code"
	@echo "  lint    - Run linter"
	@echo "  help    - Show this help message"
