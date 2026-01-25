.PHONY: help docker-build docker-up docker-down docker-restart docker-logs docker-ps docker-clean setup

# Default environment file
ENV_FILE ?= .env.dev

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ==============================================================================
# Project Setup
# ==============================================================================

setup: ## Initialize project for development (run after cloning)
	@echo "🚀 Setting up project for development..."
	@echo "  → Configuring git hooks..."
	@git config core.hooksPath .husky
	@echo "  → Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "  → Installing backend development dependencies..."
	@cd backend && go mod download
	@echo ""
	@echo "✅ Project setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Copy .env.dev to .env and configure if needed"
	@echo "  2. Run 'make dev' to start the development environment"

# ==============================================================================
# Development Commands
# ==============================================================================

dev: ## Start development environment
	@echo "Starting development environment..."
	@cp -n .env.dev .env 2>/dev/null || true
	$(MAKE) docker-up ENV_FILE=.env.dev

dev-logs: ## Show all logs in development
	docker compose logs -f

dev-rebuild: ## Rebuild and restart development environment
	@echo "Rebuilding development environment..."
	$(MAKE) docker-up-build ENV_FILE=.env.dev

# ==============================================================================
# Production Commands
# ==============================================================================

prod-setup: ## Setup production environment file
	@if [ ! -f .env ]; then \
		cp .env.docker .env; \
		echo "Created .env file. Please edit it and set secure passwords!"; \
		echo "IMPORTANT: Change DB_PASSWORD, JWT_SECRET, and other secrets"; \
	else \
		echo ".env file already exists"; \
	fi

prod: ## Start production environment
	@if [ ! -f .env ]; then \
		echo "ERROR: .env file not found. Run 'make prod-setup' first"; \
		exit 1; \
	fi
	@echo "Starting production environment..."
	$(MAKE) docker-up-build ENV_FILE=.env

# ==============================================================================
# Health & Monitoring
# ==============================================================================

health: ## Check health of all services
	@echo "Checking service health..."
	@echo "\n=== Backend Health ==="
	@curl -sf http://localhost:8080/health || echo "Backend is not responding"
	@echo "\n=== Frontend Health ==="
	@curl -sf http://localhost:3000/health || echo "Frontend is not responding"
	@echo "\n=== PostgreSQL Health ==="
	@docker compose exec -T postgres pg_isready -U postgres || echo "PostgreSQL is not ready"
	@echo "\n=== Redis Health ==="
	@docker compose exec -T redis redis-cli ping || echo "Redis is not responding"

stats: ## Show Docker resource usage
	docker stats --no-stream

inspect-network: ## Inspect Docker network
	docker network inspect erp-network

# ==============================================================================
# Local Test Environment (for E2E debugging)
# ==============================================================================

local-test-start: ## Start local test environment (DB in Docker, backend/frontend locally)
	@./docker/local-test.sh start

local-test-stop: ## Stop local test environment
	@./docker/local-test.sh stop

local-test-status: ## Show local test environment status
	@./docker/local-test.sh status

local-test-logs-backend: ## Tail backend logs
	@./docker/local-test.sh logs-backend

local-test-logs-frontend: ## Tail frontend logs
	@./docker/local-test.sh logs-frontend

local-test-e2e: ## Run E2E tests against local environment
	@./docker/local-test.sh run-e2e

local-test-e2e-ui: ## Run E2E tests with Playwright UI
	@./docker/local-test.sh run-e2e-ui

local-test-e2e-debug: ## Run E2E tests in debug mode
	@./docker/local-test.sh run-e2e-debug

local-test-clean: ## Clean local test environment
	@./docker/local-test.sh clean

# ==============================================================================
# Utility Commands
# ==============================================================================

shell-backend: ## Access backend container shell
	docker compose exec backend sh

shell-frontend: ## Access frontend container shell
	docker compose exec frontend sh

version: ## Show versions of all components
	@echo "=== Docker Version ==="
	@docker --version
	@echo "\n=== Docker Compose Version ==="
	@docker compose --version
	@echo "\n=== Application Versions ==="
	@docker compose exec backend /app/server --version 2>/dev/null || echo "Backend not running"

config: ## Show Docker Compose configuration
	docker compose config

.DEFAULT_GOAL := help
