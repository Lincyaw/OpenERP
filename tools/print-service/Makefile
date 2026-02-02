# Print Service Makefile
# Cross-platform build and installation targets

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build settings
GO ?= go
GOFLAGS ?= -trimpath
LDFLAGS := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.GitCommit=$(GIT_COMMIT)'

# Output directories
DIST_DIR := dist
BIN_NAME := print-service

# Supported platforms
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	@echo "Building for current platform..."
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BIN_NAME) ./cmd/print-service

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} ; \
		output="$(DIST_DIR)/$(BIN_NAME)-$$GOOS-$$GOARCH" ; \
		if [ "$$GOOS" = "windows" ]; then output="$$output.exe"; fi ; \
		echo "Building $$output..." ; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $$output ./cmd/print-service ; \
	done
	@echo "Build complete. Binaries in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

# Build for specific platform
.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building for Linux amd64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BIN_NAME)-linux-amd64 ./cmd/print-service

.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building for Linux arm64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BIN_NAME)-linux-arm64 ./cmd/print-service

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building for macOS amd64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BIN_NAME)-darwin-amd64 ./cmd/print-service

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building for macOS arm64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BIN_NAME)-darwin-arm64 ./cmd/print-service

.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building for Windows amd64..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BIN_NAME)-windows-amd64.exe ./cmd/print-service

# Package for distribution
.PHONY: package
package: build-all
	@echo "Packaging for distribution..."
	@mkdir -p $(DIST_DIR)/packages
	@# Linux amd64 package
	@mkdir -p $(DIST_DIR)/packages/print-service-linux-amd64
	@cp $(DIST_DIR)/$(BIN_NAME)-linux-amd64 $(DIST_DIR)/packages/print-service-linux-amd64/$(BIN_NAME)
	@cp scripts/install.sh $(DIST_DIR)/packages/print-service-linux-amd64/
	@cp configs/config.example.yaml $(DIST_DIR)/packages/print-service-linux-amd64/
	@cd $(DIST_DIR)/packages && tar -czvf print-service-linux-amd64-$(VERSION).tar.gz print-service-linux-amd64
	@# Linux arm64 package
	@mkdir -p $(DIST_DIR)/packages/print-service-linux-arm64
	@cp $(DIST_DIR)/$(BIN_NAME)-linux-arm64 $(DIST_DIR)/packages/print-service-linux-arm64/$(BIN_NAME)
	@cp scripts/install.sh $(DIST_DIR)/packages/print-service-linux-arm64/
	@cp configs/config.example.yaml $(DIST_DIR)/packages/print-service-linux-arm64/
	@cd $(DIST_DIR)/packages && tar -czvf print-service-linux-arm64-$(VERSION).tar.gz print-service-linux-arm64
	@# macOS Intel package
	@mkdir -p $(DIST_DIR)/packages/print-service-darwin-amd64
	@cp $(DIST_DIR)/$(BIN_NAME)-darwin-amd64 $(DIST_DIR)/packages/print-service-darwin-amd64/$(BIN_NAME)
	@cp scripts/install-mac.sh $(DIST_DIR)/packages/print-service-darwin-amd64/install.sh
	@cp configs/config.example.yaml $(DIST_DIR)/packages/print-service-darwin-amd64/
	@cd $(DIST_DIR)/packages && tar -czvf print-service-darwin-amd64-$(VERSION).tar.gz print-service-darwin-amd64
	@# macOS ARM package
	@mkdir -p $(DIST_DIR)/packages/print-service-darwin-arm64
	@cp $(DIST_DIR)/$(BIN_NAME)-darwin-arm64 $(DIST_DIR)/packages/print-service-darwin-arm64/$(BIN_NAME)
	@cp scripts/install-mac.sh $(DIST_DIR)/packages/print-service-darwin-arm64/install.sh
	@cp configs/config.example.yaml $(DIST_DIR)/packages/print-service-darwin-arm64/
	@cd $(DIST_DIR)/packages && tar -czvf print-service-darwin-arm64-$(VERSION).tar.gz print-service-darwin-arm64
	@# Windows package
	@mkdir -p $(DIST_DIR)/packages/print-service-windows-amd64
	@cp $(DIST_DIR)/$(BIN_NAME)-windows-amd64.exe $(DIST_DIR)/packages/print-service-windows-amd64/$(BIN_NAME).exe
	@cp scripts/install.ps1 $(DIST_DIR)/packages/print-service-windows-amd64/
	@cp configs/config.example.yaml $(DIST_DIR)/packages/print-service-windows-amd64/
	@cd $(DIST_DIR)/packages && zip -r print-service-windows-amd64-$(VERSION).zip print-service-windows-amd64
	@# Generate checksums
	@cd $(DIST_DIR)/packages && sha256sum *.tar.gz *.zip > checksums.txt
	@echo "Packages created in $(DIST_DIR)/packages/"
	@ls -la $(DIST_DIR)/packages/*.tar.gz $(DIST_DIR)/packages/*.zip $(DIST_DIR)/packages/checksums.txt 2>/dev/null || true

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Check code quality
.PHONY: lint
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w .

# Verify dependencies
.PHONY: verify
verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify
	$(GO) mod tidy

# Install locally (for development)
.PHONY: install
install: build
	@echo "Installing to /usr/local/bin..."
	@sudo cp $(DIST_DIR)/$(BIN_NAME) /usr/local/bin/
	@echo "Installed $(BIN_NAME) to /usr/local/bin/"

# Print version info
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Run the service (for development)
.PHONY: run
run:
	@echo "Running print service..."
	$(GO) run ./cmd/print-service -config configs/config.yaml

# Help
.PHONY: help
help:
	@echo "Print Service Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          Build for current platform"
	@echo "  make build-all      Build for all platforms (linux, darwin, windows)"
	@echo "  make package        Build and package for distribution"
	@echo "  make clean          Clean build artifacts"
	@echo "  make test           Run tests"
	@echo "  make test-coverage  Run tests with coverage report"
	@echo "  make lint           Run linters"
	@echo "  make fmt            Format code"
	@echo "  make verify         Verify dependencies"
	@echo "  make install        Install to /usr/local/bin"
	@echo "  make version        Print version info"
	@echo "  make run            Run the service (development)"
	@echo ""
	@echo "Platform-specific builds:"
	@echo "  make build-linux-amd64"
	@echo "  make build-darwin-amd64"
	@echo "  make build-darwin-arm64"
	@echo "  make build-windows-amd64"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION      Override version (default: git describe)"
	@echo "  BUILD_TIME   Override build time (default: current time)"
	@echo "  GIT_COMMIT   Override git commit (default: git rev-parse)"
