# ERP Backend Makefile
# Provides common development tasks including testing and coverage

.PHONY: help build test test-unit test-integration test-coverage test-coverage-html test-race \
        lint fmt clean run migrate-up migrate-down migrate-create

# Default target
help:
	@echo "ERP Backend - Available targets:"
	@echo ""
	@echo "  build              Build the application"
	@echo "  run                Run the server"
	@echo ""
	@echo "  test               Run all tests"
	@echo "  test-unit          Run unit tests only"
	@echo "  test-race          Run tests with race detector"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  test-coverage-html Generate HTML coverage report"
	@echo "  test-coverage-ci   Generate coverage for CI (with threshold check)"
	@echo ""
	@echo "  lint               Run linter"
	@echo "  fmt                Format code"
	@echo ""
	@echo "  migrate-up         Run database migrations"
	@echo "  migrate-down       Rollback last migration"
	@echo "  migrate-create     Create a new migration (NAME=migration_name)"
	@echo ""
	@echo "  clean              Clean build artifacts"

# Build variables
BINARY_NAME=erp-server
MIGRATE_BINARY=migrate
BUILD_DIR=bin
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html

# Go variables
GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOFMT=gofmt
GOVET=$(GO) vet

# Packages
PACKAGES=./...
UNIT_PACKAGES=./internal/...

# Coverage threshold (80%)
COVERAGE_THRESHOLD=80

# Build targets
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

build-migrate:
	@echo "Building migration CLI..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(MIGRATE_BINARY) ./cmd/migrate

run: build
	@echo "Starting server..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Test targets
test:
	@echo "Running all tests..."
	$(GOTEST) -v $(PACKAGES)

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short $(UNIT_PACKAGES)

test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -race -v $(PACKAGES)

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(PACKAGES)
	@echo ""
	@echo "Coverage Summary:"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | tail -n 1
	@echo ""
	@echo "Package Coverage:"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | grep -E "^github.com/erp/backend/internal" | head -20

test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-coverage-ci:
	@echo "Running coverage for CI with threshold check..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(PACKAGES)
	@echo ""
	@COVERAGE=$$($(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "ERROR: Coverage $$COVERAGE% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "OK: Coverage $$COVERAGE% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

# Run tests for a specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-pkg PKG=./internal/..."; \
		exit 1; \
	fi
	$(GOTEST) -v -cover $(PKG)

# Lint and format
lint:
	@echo "Running linter..."
	$(GOVET) $(PACKAGES)
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt'"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi

# Migration targets
migrate-up: build-migrate
	@echo "Running migrations..."
	./$(BUILD_DIR)/$(MIGRATE_BINARY) up

migrate-down: build-migrate
	@echo "Rolling back last migration..."
	./$(BUILD_DIR)/$(MIGRATE_BINARY) step -1

migrate-create: build-migrate
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	./$(BUILD_DIR)/$(MIGRATE_BINARY) create $(NAME)

migrate-status: build-migrate
	@echo "Checking migration status..."
	./$(BUILD_DIR)/$(MIGRATE_BINARY) version

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)

# CI target - runs all checks
ci: fmt-check lint test-coverage-ci
	@echo "All CI checks passed!"

# Development helpers
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development dependencies installed."
