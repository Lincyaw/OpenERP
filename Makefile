# =============================================================================
# ERP System - Makefile
# =============================================================================
# Unified commands for Docker mode and local development mode.
# Run 'make help' to see all available commands.
# =============================================================================

.PHONY: help setup docker-up docker-down docker-logs docker-build \
        dev dev-stop dev-backend dev-frontend dev-status \
        db-migrate db-seed db-reset db-psql \
        e2e e2e-ui e2e-debug e2e-local \
        otel-up otel-down otel-logs otel-status \
        clean logs api-docs

# Default target
.DEFAULT_GOAL := help

# Configuration
DOCKER_COMPOSE := docker compose
MIGRATE_IMAGE := migrate/migrate:v4.17.0
PLAYWRIGHT_IMAGE := mcr.microsoft.com/playwright:v1.58.0-noble

# Colors for output
CYAN := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

# =============================================================================
# Help
# =============================================================================

help: ## Show this help message
	@echo ""
	@echo "$(CYAN)ERP System - Development Commands$(NC)"
	@echo ""
	@echo "$(GREEN)Setup:$(NC)"
	@grep -E '^setup:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Docker Mode (all services in containers):$(NC)"
	@grep -E '^docker-.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Local Development (database in Docker, app locally):$(NC)"
	@grep -E '^dev.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Database Management:$(NC)"
	@grep -E '^db-.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)E2E Testing:$(NC)"
	@grep -E '^e2e.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Observability (OpenTelemetry):$(NC)"
	@grep -E '^otel-.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Other:$(NC)"
	@grep -E '^(clean|logs|api-docs):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

# =============================================================================
# Setup
# =============================================================================

setup: ## Initialize project (run after git clone)
	@echo "$(CYAN)Setting up project...$(NC)"
	@if [ ! -f .env ]; then \
		echo "  → Creating .env from .env.example..."; \
		cp .env.example .env; \
	fi
	@echo "  → Configuring git hooks..."
	@git config core.hooksPath .husky 2>/dev/null || true
	@echo "  → Installing frontend dependencies..."
	@cd frontend && npm install
	@echo "  → Downloading backend dependencies..."
	@cd backend && go mod download
	@echo ""
	@echo "$(GREEN)Setup complete!$(NC)"
	@echo ""
	@echo "Next steps:"
	@echo "  $(CYAN)make docker-up$(NC)      # Run all services in Docker"
	@echo "  $(CYAN)make dev$(NC)            # Start database for local development"

# =============================================================================
# Docker Mode (Full Stack in Containers)
# =============================================================================

docker-up: ## Start all services in Docker (postgres, redis, backend, frontend)
	@echo "$(CYAN)Starting all services in Docker...$(NC)"
	@$(DOCKER_COMPOSE) up -d postgres redis
	@echo "  → Waiting for database..."
	@sleep 5
	@$(MAKE) db-migrate
	@$(MAKE) db-seed
	@$(DOCKER_COMPOSE) --profile docker up -d
	@echo ""
	@echo "$(GREEN)All services started!$(NC)"
	@echo "  Frontend:  http://localhost:$${FRONTEND_PORT:-3000}"
	@echo "  Backend:   http://localhost:$${BACKEND_PORT:-8080}"
	@echo "  Login:     admin / admin123"

docker-down: ## Stop all Docker services
	@echo "$(CYAN)Stopping all services...$(NC)"
	@$(DOCKER_COMPOSE) --profile docker --profile e2e down
	@echo "$(GREEN)Services stopped.$(NC)"

# =============================================================================
# Local Development Mode (Database in Docker, App Locally)
# =============================================================================

dev: ## Start database services (postgres + redis) for local development
	@echo "$(CYAN)Starting database services...$(NC)"
	@$(DOCKER_COMPOSE) up -d postgres redis
	@echo "  → Waiting for database..."
	@sleep 5
	@$(MAKE) db-migrate
	@$(MAKE) db-seed
	@echo ""
	@echo "$(GREEN)Database ready!$(NC)"
	@echo "  PostgreSQL: localhost:$${DB_PORT:-5432}"
	@echo "  Redis:      localhost:$${REDIS_PORT:-6379}"
	@echo ""
	@echo "Next steps:"
	@echo "  $(CYAN)make dev-backend$(NC)    # Run backend locally"
	@echo "  $(CYAN)make dev-frontend$(NC)   # Run frontend locally"

dev-stop: ## Stop database services
	@echo "$(CYAN)Stopping database services...$(NC)"
	@$(DOCKER_COMPOSE) stop postgres redis
	@echo "$(GREEN)Database stopped.$(NC)"

dev-backend: ## Run backend locally (requires database)
	@echo "$(CYAN)Starting backend at http://localhost:$${BACKEND_PORT:-8080}$(NC)"
	@cd backend && go run cmd/server/main.go

dev-frontend: ## Run frontend locally
	@echo "$(CYAN)Starting frontend at http://localhost:$${FRONTEND_PORT:-3000}$(NC)"
	@cd frontend && VITE_API_BASE_URL="http://localhost:$${BACKEND_PORT:-8080}/api/v1" npm run dev -- --host --port $${FRONTEND_PORT:-3000}

# =============================================================================
# Database Management
# =============================================================================

db-migrate: ## Run database migrations
	@echo "$(CYAN)Running database migrations...$(NC)"
	@docker run --rm --network erp-network \
		-v "$(PWD)/backend/migrations:/migrations:ro" \
		$(MIGRATE_IMAGE) \
		-path=/migrations \
		-database "postgres://$${DB_USER:-postgres}:$${DB_PASSWORD:-admin123}@erp-postgres:5432/$${DB_NAME:-erp_dev}?sslmode=disable" \
		up 2>/dev/null || echo "  Migrations may already be applied"
	@echo "$(GREEN)Migrations complete.$(NC)"

db-seed: ## Load seed data into database
	@echo "$(CYAN)Loading seed data...$(NC)"
	@docker exec -i erp-postgres psql -U $${DB_USER:-postgres} -d $${DB_NAME:-erp_dev} < docker/seed-data.sql 2>/dev/null \
		&& echo "$(GREEN)Seed data loaded.$(NC)" \
		|| echo "$(YELLOW)Seed data may already be loaded (conflicts ignored).$(NC)"

db-reset: ## Reset database (drop data, run migrations, seed)
	@echo "$(CYAN)Resetting database...$(NC)"
	@$(DOCKER_COMPOSE) stop postgres
	@$(DOCKER_COMPOSE) rm -f postgres
	@docker volume rm erp-postgres-data 2>/dev/null || true
	@$(DOCKER_COMPOSE) up -d postgres
	@echo "  → Waiting for database..."
	@sleep 5
	@$(MAKE) db-migrate
	@$(MAKE) db-seed
	@echo "$(GREEN)Database reset complete.$(NC)"

db-psql: ## Open psql shell to database
	@docker exec -it erp-postgres psql -U $${DB_USER:-postgres} -d $${DB_NAME:-erp_dev}

# =============================================================================
# E2E Testing
# =============================================================================

e2e: ## Run E2E tests (resets environment, runs all tests)
	@echo "$(CYAN)Running E2E tests with fresh environment...$(NC)"
	@$(MAKE) db-reset
	@echo ""
	@echo "$(CYAN)Starting backend and frontend...$(NC)"
	@$(DOCKER_COMPOSE) --profile docker up -d --build
	@echo "  → Waiting for services to be healthy..."
	@sleep 10
	@echo ""
	@echo "$(CYAN)Running Playwright tests...$(NC)"
	@docker run --rm \
		--user "$$(id -u):$$(id -g)" \
		--network erp-network \
		-v "$(PWD)/frontend:/app" \
		-w /app \
		-e HOME=/tmp \
		-e E2E_BASE_URL="http://erp-frontend:80" \
		-e CI=false \
		$(PLAYWRIGHT_IMAGE) \
		npx playwright test --reporter=list $(ARGS)
	@echo ""
	@echo "$(GREEN)E2E tests complete.$(NC)"

e2e-ui: ## Run E2E tests with Playwright UI (requires local services)
	@echo "$(CYAN)Starting Playwright UI...$(NC)"
	@cd frontend && E2E_BASE_URL="http://localhost:$${FRONTEND_PORT:-3000}" npx playwright test --ui

e2e-debug: ## Run E2E tests in debug mode (requires local services)
	@echo "$(CYAN)Starting Playwright in debug mode...$(NC)"
	@cd frontend && E2E_BASE_URL="http://localhost:$${FRONTEND_PORT:-3000}" npx playwright test --debug $(ARGS)

e2e-local: ## Run E2E tests locally (no Docker, requires running services)
	@echo "$(CYAN)Running E2E tests against local services...$(NC)"
	@cd frontend && E2E_BASE_URL="http://localhost:$${FRONTEND_PORT:-3000}" npx playwright test --reporter=list $(ARGS)

# =============================================================================
# Other Commands
# =============================================================================

clean: ## Stop all services and remove data
	@echo "$(CYAN)Cleaning up...$(NC)"
	@$(DOCKER_COMPOSE) --profile docker --profile e2e --profile migrate --profile otel down -v
	@docker volume rm erp-postgres-data erp-redis-data erp-otel-logs 2>/dev/null || true
	@rm -rf logs/ bin/
	@echo "$(GREEN)Cleanup complete.$(NC)"

# =============================================================================
# Observability (OpenTelemetry)
# =============================================================================

otel-up: ## Start OpenTelemetry Collector
	@echo "$(CYAN)Starting OpenTelemetry Collector...$(NC)"
	@$(DOCKER_COMPOSE) --profile otel up -d otel-collector
	@echo ""
	@echo "$(GREEN)OTEL Collector started!$(NC)"
	@echo "  gRPC endpoint:   localhost:$${OTEL_GRPC_PORT:-4317}"
	@echo "  HTTP endpoint:   localhost:$${OTEL_HTTP_PORT:-4318}"
	@echo "  Health check:    http://localhost:13133/health"
	@echo "  Metrics:         http://localhost:8888/metrics"
	@echo "  zpages:          http://localhost:55679/debug/tracez"

otel-down: ## Stop OpenTelemetry Collector
	@echo "$(CYAN)Stopping OpenTelemetry Collector...$(NC)"
	@$(DOCKER_COMPOSE) stop otel-collector
	@echo "$(GREEN)OTEL Collector stopped.$(NC)"

otel-logs: ## View OpenTelemetry Collector logs
	@$(DOCKER_COMPOSE) logs -f otel-collector

otel-status: ## Show OTEL Collector status and health
	@echo "$(CYAN)OTEL Collector Status:$(NC)"
	@docker inspect erp-otel-collector --format '  Container: {{.State.Status}}' 2>/dev/null || echo "  Container: Not running"
	@echo ""
	@curl -s http://localhost:13133/health 2>/dev/null | jq -r '"  Health: " + .status' 2>/dev/null || echo "  Health: Unavailable (collector not running or port blocked)"
	@echo ""
	@echo "$(CYAN)Endpoints:$(NC)"
	@echo "  gRPC:    localhost:$${OTEL_GRPC_PORT:-4317}"
	@echo "  HTTP:    localhost:$${OTEL_HTTP_PORT:-4318}"
	@echo "  Health:  http://localhost:13133/health"
	@echo "  Metrics: http://localhost:8888/metrics"
	@echo ""
	@echo "$(CYAN)Data volume:$(NC)"
	@docker volume inspect erp-otel-logs --format '  {{.Mountpoint}}' 2>/dev/null || echo "  Volume not created yet"


