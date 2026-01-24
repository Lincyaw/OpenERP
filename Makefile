.PHONY: help docker-build docker-up docker-down docker-restart docker-logs docker-ps docker-clean

# Default environment file
ENV_FILE ?= .env.dev

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ==============================================================================
# Docker Commands
# ==============================================================================

docker-build: ## Build all Docker images
	@echo "Building Docker images..."
	docker-compose build

docker-up: ## Start all services
	@echo "Starting services..."
	docker-compose --env-file $(ENV_FILE) up -d
	@echo "Services started. Check status with: make docker-ps"

docker-up-build: ## Build and start all services
	@echo "Building and starting services..."
	docker-compose --env-file $(ENV_FILE) up -d --build

docker-down: ## Stop all services
	@echo "Stopping services..."
	docker-compose down

docker-down-volumes: ## Stop all services and remove volumes (WARNING: deletes data)
	@echo "WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
	fi

docker-restart: ## Restart all services
	@echo "Restarting services..."
	docker-compose restart

docker-restart-backend: ## Restart backend service
	@echo "Restarting backend..."
	docker-compose restart backend

docker-restart-frontend: ## Restart frontend service
	@echo "Restarting frontend..."
	docker-compose restart frontend

docker-logs: ## Show logs from all services
	docker-compose logs -f

docker-logs-backend: ## Show backend logs
	docker-compose logs -f backend

docker-logs-frontend: ## Show frontend logs
	docker-compose logs -f frontend

docker-logs-postgres: ## Show PostgreSQL logs
	docker-compose logs -f postgres

docker-logs-redis: ## Show Redis logs
	docker-compose logs -f redis

docker-ps: ## Show running services
	docker-compose ps

docker-clean: ## Remove stopped containers and unused images
	@echo "Cleaning up Docker resources..."
	docker-compose down
	docker system prune -f
	@echo "Cleanup complete"

# ==============================================================================
# Database Commands
# ==============================================================================

db-shell: ## Access PostgreSQL shell
	docker-compose exec postgres psql -U postgres -d erp

db-migrate-up: ## Run database migrations
	docker-compose run --rm migrate \
		-path=/migrations \
		-database "postgres://postgres:postgres123@postgres:5432/erp?sslmode=disable" \
		up

db-migrate-down: ## Rollback last migration
	docker-compose run --rm migrate \
		-path=/migrations \
		-database "postgres://postgres:postgres123@postgres:5432/erp?sslmode=disable" \
		down 1

db-migrate-version: ## Show current migration version
	docker-compose run --rm migrate \
		-path=/migrations \
		-database "postgres://postgres:postgres123@postgres:5432/erp?sslmode=disable" \
		version

db-backup: ## Backup database to file
	@echo "Creating database backup..."
	@mkdir -p backups
	docker-compose exec -T postgres pg_dump -U postgres erp > backups/backup-$$(date +%Y%m%d-%H%M%S).sql
	@echo "Backup created in backups/ directory"

db-restore: ## Restore database from latest backup
	@echo "Restoring database from latest backup..."
	@LATEST=$$(ls -t backups/*.sql | head -1); \
	if [ -z "$$LATEST" ]; then \
		echo "No backup files found"; \
		exit 1; \
	fi; \
	echo "Restoring from: $$LATEST"; \
	docker-compose exec -T postgres psql -U postgres erp < $$LATEST; \
	echo "Database restored"

# ==============================================================================
# Redis Commands
# ==============================================================================

redis-shell: ## Access Redis CLI
	docker-compose exec redis redis-cli

redis-monitor: ## Monitor Redis commands
	docker-compose exec redis redis-cli monitor

redis-info: ## Show Redis info
	docker-compose exec redis redis-cli info

redis-flush: ## Flush all Redis keys (WARNING: deletes all cache)
	@echo "WARNING: This will delete all cached data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose exec redis redis-cli FLUSHALL; \
		echo "Redis flushed"; \
	fi

# ==============================================================================
# Development Commands
# ==============================================================================

dev: ## Start development environment
	@echo "Starting development environment..."
	@cp -n .env.dev .env 2>/dev/null || true
	$(MAKE) docker-up ENV_FILE=.env.dev

dev-logs: ## Show all logs in development
	docker-compose logs -f

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
	@docker-compose exec -T postgres pg_isready -U postgres || echo "PostgreSQL is not ready"
	@echo "\n=== Redis Health ==="
	@docker-compose exec -T redis redis-cli ping || echo "Redis is not responding"

stats: ## Show Docker resource usage
	docker stats --no-stream

inspect-network: ## Inspect Docker network
	docker network inspect erp-network

# ==============================================================================
# Testing Environment
# ==============================================================================

test-env-up: ## Start fresh test environment with clean database
	@echo "Starting fresh test environment..."
	@docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
	@docker-compose -f docker-compose.test.yml up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Test environment is ready on ports: Frontend=3001, Backend=8081"

test-env-down: ## Stop and cleanup test environment
	@echo "Stopping test environment..."
	@docker-compose -f docker-compose.test.yml down -v
	@echo "Test environment stopped and cleaned"

test-seed: ## Load test data into test database
	@echo "Loading test data..."
	@docker-compose -f docker-compose.test.yml exec -T postgres psql -U postgres -d erp_test < docker/seed-data.sql
	@echo "Test data loaded successfully"

test-seed-api: ## Load test data via API calls
	@echo "Loading test data via API..."
	@bash docker/seed-api.sh http://localhost:8081/api/v1
	@echo "API test data loaded successfully"

test-load: ## Run load tests against test environment
	@echo "Running load tests..."
	@bash docker/load-test.sh http://localhost:8081/api/v1
	@echo "Load tests completed"

test-quick: ## Quick API smoke test
	@echo "Running quick API tests..."
	@bash docker/quick-test.sh http://localhost:8081/api/v1

test-full: ## Full test cycle: start env, seed data, load test, stop
	@echo "Starting full test cycle..."
	$(MAKE) test-env-up
	@sleep 5
	$(MAKE) test-seed-api
	@sleep 2
	$(MAKE) test-quick
	@echo ""
	@echo "Test environment is ready for load testing"
	@echo "Run 'make test-load' to perform load tests"
	@echo "Run 'make test-env-down' to cleanup when done"

test-reset: ## Reset test database and reload seed data
	@echo "Resetting test database..."
	@docker-compose -f docker-compose.test.yml exec postgres psql -U postgres -c "DROP DATABASE IF EXISTS erp_test;"
	@docker-compose -f docker-compose.test.yml exec postgres psql -U postgres -c "CREATE DATABASE erp_test;"
	@sleep 2
	@docker-compose -f docker-compose.test.yml restart backend
	@sleep 5
	$(MAKE) test-seed-api
	@echo "Test database reset complete"

test-logs: ## Show test environment logs
	@docker-compose -f docker-compose.test.yml logs -f

test-shell: ## Access test database shell
	@docker-compose -f docker-compose.test.yml exec postgres psql -U postgres -d erp_test

# Legacy test command (kept for backwards compatibility)
test-api: ## Test API endpoints
	@echo "Testing API health..."
	curl -v http://localhost:8080/health
	@echo "\n\nTesting API ping..."
	curl -v http://localhost:8080/api/v1/ping

# ==============================================================================
# Utility Commands
# ==============================================================================

shell-backend: ## Access backend container shell
	docker-compose exec backend sh

shell-frontend: ## Access frontend container shell
	docker-compose exec frontend sh

version: ## Show versions of all components
	@echo "=== Docker Version ==="
	@docker --version
	@echo "\n=== Docker Compose Version ==="
	@docker-compose --version
	@echo "\n=== Application Versions ==="
	@docker-compose exec backend /app/server --version 2>/dev/null || echo "Backend not running"

config: ## Show Docker Compose configuration
	docker-compose config

.DEFAULT_GOAL := help
