# kubectl-fluid-inspect Makefile

BINARY_NAME=kubectl-fluid
VERSION?=dev
BUILD_DIR=bin
GO_FLAGS=-ldflags "-X github.com/mrhapile/kubectl-fluid-inspect/pkg/cmd.Version=$(VERSION)"

.PHONY: all build clean test lint install

all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/kubectl-fluid/

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

## test: Run unit tests
test:
	@echo "Running tests..."
	go test -v ./...

## lint: Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run ./...

## install: Install the binary to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

## uninstall: Remove the binary from /usr/local/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'

# Development helpers
.PHONY: fmt vet deps

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
