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
        pyroscope-up pyroscope-down pyroscope-ui pyroscope-logs pyroscope-status \
        loadgen-build loadgen-clean loadgen-test loadgen-run loadgen-stress \
        loadgen-scenario loadgen-dry-run loadgen-list loadgen-validate \
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
	@echo "$(GREEN)Profiling (Pyroscope):$(NC)"
	@grep -E '^pyroscope-.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Load Generator:$(NC)"
	@grep -E '^loadgen-.*:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(NC) %s\n", $$1, $$2}'
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
	@$(DOCKER_COMPOSE) --profile otel up -d postgres redis otel-collector
	@echo "  → Waiting for database..."
	@sleep 5
	@$(MAKE) db-migrate
	@$(MAKE) db-seed
	@$(DOCKER_COMPOSE) --profile docker up -d $(ARGS)
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
	@cd frontend &&  npm run dev -- --host --port $${FRONTEND_PORT:-3000}

gen:
	$(MAKE) -C backend docs
	@cd frontend && npx orval


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
	@echo "$(CYAN)Initializing storage bucket...$(NC)"
	@docker run --rm --network erp-network \
		-e AWS_ACCESS_KEY_ID=$${RUSTFS_ACCESS_KEY:-rustfsadmin} \
		-e AWS_SECRET_ACCESS_KEY=$${RUSTFS_SECRET_KEY:-rustfsadmin123} \
		amazon/aws-cli:latest \
		--endpoint-url http://erp-rustfs:9000 \
		s3 mb s3://$${ERP_STORAGE_BUCKET:-erp-attachments} 2>/dev/null || echo "  Bucket already exists"
	@echo "$(GREEN)Storage initialized.$(NC)"

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
	@$(DOCKER_COMPOSE) --profile docker up -d
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

# =============================================================================
# Other Commands
# =============================================================================

clean: ## Stop all services and remove data (with orphan cleanup)
	@echo "$(CYAN)Cleaning up...$(NC)"
	@$(DOCKER_COMPOSE) --profile docker --profile e2e --profile migrate --profile otel down -v --remove-orphans
	@echo "  → Removing orphan containers..."
	@docker container rm -f erp-backend erp-frontend erp-pyroscope erp-otel-collector erp-migrate erp-postgres erp-redis erp-playwright 2>/dev/null || true
	@echo "  → Removing erp-network..."
	@docker network rm erp-network 2>/dev/null || true
	@echo "  → Removing volumes..."
	@docker volume rm erp-postgres-data erp-redis-data erp-otel-logs erp-pyroscope-data 2>/dev/null || true
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
	@echo "  gRPC endpoint:   localhost:$${OTEL_GRPC_PORT:-14317}"
	@echo "  HTTP endpoint:   localhost:$${OTEL_HTTP_PORT:-14318}"
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
	@echo "  gRPC:    localhost:$${OTEL_GRPC_PORT:-14317}"
	@echo "  HTTP:    localhost:$${OTEL_HTTP_PORT:-14318}"
	@echo "  Health:  http://localhost:13133/health"
	@echo "  Metrics: http://localhost:8888/metrics"
	@echo ""
	@echo "$(CYAN)Data volume:$(NC)"
	@docker volume inspect erp-otel-logs --format '  {{.Mountpoint}}' 2>/dev/null || echo "  Volume not created yet"

# =============================================================================
# Profiling (Pyroscope)
# =============================================================================

pyroscope-up: ## Start Pyroscope profiling server
	@echo "$(CYAN)Starting Pyroscope...$(NC)"
	@$(DOCKER_COMPOSE) --profile otel up -d pyroscope
	@echo ""
	@echo "$(GREEN)Pyroscope started!$(NC)"
	@echo "  UI:       http://localhost:$${PYROSCOPE_PORT:-4040}"
	@echo "  API:      http://localhost:$${PYROSCOPE_PORT:-4040}/api"
	@echo ""
	@echo "Backend connection string: http://pyroscope:4040"

pyroscope-down: ## Stop Pyroscope profiling server
	@echo "$(CYAN)Stopping Pyroscope...$(NC)"
	@$(DOCKER_COMPOSE) stop pyroscope
	@echo "$(GREEN)Pyroscope stopped.$(NC)"

pyroscope-ui: ## Open Pyroscope UI in browser
	@echo "$(CYAN)Opening Pyroscope UI...$(NC)"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open "http://localhost:$${PYROSCOPE_PORT:-4040}" || \
	command -v open >/dev/null 2>&1 && open "http://localhost:$${PYROSCOPE_PORT:-4040}" || \
	echo "Please open http://localhost:$${PYROSCOPE_PORT:-4040} in your browser"

pyroscope-logs: ## View Pyroscope logs
	@$(DOCKER_COMPOSE) logs -f pyroscope

pyroscope-status: ## Show Pyroscope status and health
	@echo "$(CYAN)Pyroscope Status:$(NC)"
	@docker inspect erp-pyroscope --format '  Container: {{.State.Status}}' 2>/dev/null || echo "  Container: Not running"
	@echo ""
	@curl -s "http://localhost:$${PYROSCOPE_PORT:-4040}/ready" >/dev/null 2>&1 && echo "  Health: Ready" || echo "  Health: Not ready (pyroscope not running or port blocked)"
	@echo ""
	@echo "$(CYAN)Endpoints:$(NC)"
	@echo "  UI:      http://localhost:$${PYROSCOPE_PORT:-4040}"
	@echo "  API:     http://localhost:$${PYROSCOPE_PORT:-4040}/api"
	@echo "  Ready:   http://localhost:$${PYROSCOPE_PORT:-4040}/ready"
	@echo ""
	@echo "$(CYAN)Data volume:$(NC)"
	@docker volume inspect erp-pyroscope-data --format '  {{.Mountpoint}}' 2>/dev/null || echo "  Volume not created yet"

# =============================================================================
# Load Generator
# =============================================================================

# Build variables for loadgen
LOADGEN_DIR := tools/loadgen
LOADGEN_BIN := $(LOADGEN_DIR)/bin/loadgen
LOADGEN_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LOADGEN_BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LOADGEN_GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

loadgen-build: ## Build the load generator binary
	@echo "$(CYAN)Building load generator...$(NC)"
	@mkdir -p $(LOADGEN_DIR)/bin
	@cd $(LOADGEN_DIR) && CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(LOADGEN_VERSION) -X main.buildTime=$(LOADGEN_BUILD_TIME) -X main.gitCommit=$(LOADGEN_GIT_COMMIT)" \
		-o bin/loadgen \
		./cmd/main.go
	@echo "$(GREEN)Build complete!$(NC)"
	@echo "  Binary: $(LOADGEN_BIN)"
	@ls -lh $(LOADGEN_BIN) | awk '{print "  Size:   " $$5}'

loadgen-clean: ## Clean load generator build artifacts
	@echo "$(CYAN)Cleaning load generator...$(NC)"
	@rm -rf $(LOADGEN_DIR)/bin
	@echo "$(GREEN)Clean complete.$(NC)"

loadgen-test: ## Run load generator tests
	@echo "$(CYAN)Running load generator tests...$(NC)"
	@cd $(LOADGEN_DIR) && go test -v ./...
	@echo "$(GREEN)Tests complete.$(NC)"

loadgen-run: loadgen-build ## Run load generator with default ERP config (5m, 100 QPS)
	@echo "$(CYAN)Running load generator...$(NC)"
	@echo "  Config:   $(LOADGEN_DIR)/configs/erp.yaml"
	@echo "  Duration: 5m"
	@echo "  QPS:      100"
	@echo ""
	@$(LOADGEN_BIN) -config $(LOADGEN_DIR)/configs/erp.yaml \
		-duration 5m -qps 100 -v
	@echo ""
	@echo "$(GREEN)Load test complete.$(NC)"

loadgen-stress: loadgen-build ## Run stress test (30m, high QPS ramp-up)
	@echo "$(CYAN)Running stress test...$(NC)"
	@echo "  Config:   $(LOADGEN_DIR)/configs/erp.yaml"
	@echo "  Duration: 30m"
	@echo "  QPS:      500 (ramp up)"
	@echo ""
	@$(LOADGEN_BIN) -config $(LOADGEN_DIR)/configs/erp.yaml \
		-duration 30m -qps 500 -v
	@echo ""
	@echo "$(GREEN)Stress test complete.$(NC)"

loadgen-scenario: loadgen-build ## Run load test for specific scenario (usage: make loadgen-scenario SCENARIO=browse_catalog)
	@if [ -z "$(SCENARIO)" ]; then \
		echo "$(RED)Error: SCENARIO is required$(NC)"; \
		echo ""; \
		echo "Usage: make loadgen-scenario SCENARIO=<name>"; \
		echo ""; \
		echo "Available scenarios:"; \
		echo "  - browse_catalog      (Simulates browsing the product catalog)"; \
		echo "  - create_sales_order  (Simulates creating a sales order)"; \
		echo "  - create_purchase_order (Simulates creating a purchase order)"; \
		echo "  - check_inventory     (Simulates checking inventory status)"; \
		echo "  - review_finances     (Simulates reviewing financial status)"; \
		echo "  - view_reports        (Simulates viewing reports)"; \
		exit 1; \
	fi
	@echo "$(CYAN)Running scenario: $(SCENARIO)...$(NC)"
	@$(LOADGEN_BIN) -config $(LOADGEN_DIR)/configs/erp.yaml -v $(ARGS)
	@echo ""
	@echo "$(GREEN)Scenario test complete.$(NC)"

loadgen-dry-run: loadgen-build ## Show execution plan without running (dry-run mode)
	@echo "$(CYAN)Load generator dry run...$(NC)"
	@$(LOADGEN_BIN) -config $(LOADGEN_DIR)/configs/erp.yaml -dry-run -v

loadgen-list: loadgen-build ## List all configured endpoints
	@$(LOADGEN_BIN) -config $(LOADGEN_DIR)/configs/erp.yaml -list

loadgen-validate: loadgen-build ## Validate load generator configuration
	@echo "$(CYAN)Validating configuration...$(NC)"
	@$(LOADGEN_BIN) -config $(LOADGEN_DIR)/configs/erp.yaml -validate
	@echo "$(GREEN)Configuration valid.$(NC)"
