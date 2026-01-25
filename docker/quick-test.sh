#!/bin/bash
# =============================================================================
# ERP Quick Test Script
# =============================================================================
# Quick validation script to verify the test environment is working correctly.
# This script:
#   1. Checks if Docker services are healthy
#   2. Tests database connectivity
#   3. Runs database migrations
#   4. Seeds test data
#   5. Performs basic API health checks
#
# Usage:
#   ./docker/quick-test.sh [command]
#
# Commands:
#   start   - Start test environment and run validations (default)
#   stop    - Stop test environment
#   restart - Restart test environment
#   status  - Show status of services
#   logs    - Show logs from all services
#   seed    - Run seed data only (requires running services)
#   api     - Run API smoke tests only (requires running services)
#   clean   - Stop and remove all test data
#
# Exit codes:
#   0 - Success
#   1 - Failure
# =============================================================================

set -e

# Configuration
COMPOSE_FILE="docker-compose.test.yml"
PROJECT_NAME="erp-test"
API_URL="http://localhost:8081/api/v1"
DB_HOST="localhost"
DB_PORT="5433"
DB_NAME="erp_test"
DB_USER="postgres"
DB_PASSWORD="test123"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Working directory
cd "$(dirname "$0")/.."

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo ""
    echo -e "${YELLOW}▶ $1${NC}"
}

# Check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is required but not installed."
        return 1
    fi
}

# Wait for a service to be healthy
wait_for_service() {
    local service="$1"
    local max_attempts="${2:-2}"
    local attempt=1

    log_info "Waiting for $service to be healthy..."

    while [ $attempt -le $max_attempts ]; do
        if docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps "$service" 2>/dev/null | grep -q "healthy"; then
            log_success "$service is healthy"
            return 0
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done

    echo ""
    log_error "$service failed to become healthy after $max_attempts attempts"
    return 1
}

# Wait for database to be ready
wait_for_database() {
    local max_attempts="${1:-30}"
    local attempt=1

    log_info "Waiting for database connection..."

    while [ $attempt -le $max_attempts ]; do
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &>/dev/null; then
            log_success "Database is ready"
            return 0
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done

    echo ""
    log_error "Database connection failed after $max_attempts attempts"
    return 1
}

# Wait for API to be ready
wait_for_api() {
    local max_attempts="${1:-30}"
    local attempt=1

    log_info "Waiting for API to be ready..."

    while [ $attempt -le $max_attempts ]; do
        if curl -s "http://localhost:8081/health" &>/dev/null; then
            log_success "API is ready"
            return 0
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done

    echo ""
    log_error "API failed to respond after $max_attempts attempts"
    return 1
}

# =============================================================================
# Commands
# =============================================================================

cmd_start() {
    log_step "Starting ERP Test Environment"

    # Check prerequisites
    check_command "docker" || exit 1
    check_command "curl" || exit 1

    # Start services
    log_info "Starting Docker services..."
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d

    # Wait for services to be healthy
    log_step "Waiting for Services"
    wait_for_service "postgres" 60 || exit 1
    wait_for_service "redis" 30 || exit 1

    # Wait for migrations to complete
    log_step "Running Migrations"
    log_info "Waiting for migration container to complete..."
    sleep 5  # Give migrations time to start

    # Check migration status
    local migrate_status
    migrate_status=$(docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps migrate --format json 2>/dev/null | grep -o '"State":"[^"]*"' | cut -d'"' -f4 || echo "unknown")

    if [ "$migrate_status" = "exited" ]; then
        local exit_code
        exit_code=$(docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps migrate --format json 2>/dev/null | grep -o '"ExitCode":[0-9]*' | cut -d':' -f2 || echo "1")
        if [ "$exit_code" = "0" ]; then
            log_success "Migrations completed successfully"
        else
            log_warning "Migrations may have failed (exit code: $exit_code)"
        fi
    fi

    # Wait for backend to be healthy
    if ! wait_for_service "backend" 2; then
        log_error "Backend failed to start. Showing backend logs:"
        docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs --tail=100 backend
        exit 1
    fi

    # Seed data
    cmd_seed

    # Run quick API checks
    cmd_api_quick

    log_step "Environment Ready!"
    echo ""
    echo -e "  ${GREEN}Frontend:${NC}  http://localhost:3001"
    echo -e "  ${GREEN}Backend:${NC}   http://localhost:8081"
    echo -e "  ${GREEN}Database:${NC}  localhost:5433"
    echo -e "  ${GREEN}Redis:${NC}     localhost:6380"
    echo ""
    echo -e "  ${BLUE}Login credentials:${NC}"
    echo "    Username: admin"
    echo "    Password: test123"
    echo ""
}

cmd_stop() {
    log_step "Stopping ERP Test Environment"
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down
    log_success "Services stopped"
}

cmd_restart() {
    cmd_stop
    cmd_start
}

cmd_status() {
    log_step "Service Status"
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps
}

cmd_logs() {
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f "${@:2}"
}

cmd_seed() {
    log_step "Seeding Test Data"

    # Check if psql is available, otherwise use docker
    if command -v psql &> /dev/null; then
        log_info "Seeding data via local psql..."
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f docker/seed-data.sql 2>/dev/null && \
            log_success "Seed data applied" || \
            log_warning "Seed data may have already been applied (conflicts ignored)"
    else
        log_info "Seeding data via Docker..."
        docker exec -i erp-test-postgres psql -U "$DB_USER" -d "$DB_NAME" < docker/seed-data.sql 2>/dev/null && \
            log_success "Seed data applied" || \
            log_warning "Seed data may have already been applied (conflicts ignored)"
    fi
}

cmd_api() {
    log_step "Running Full API Smoke Tests"
    ./docker/test-api.sh "$API_URL"
}

cmd_api_quick() {
    log_step "Running Quick API Checks"

    local passed=0
    local failed=0

    # Health check
    if curl -s "http://localhost:8081/health" | grep -q "ok\|healthy" 2>/dev/null; then
        log_success "Health endpoint"
        ((passed++))
    else
        log_error "Health endpoint failed"
        ((failed++))
    fi

    # System info
    if curl -s "${API_URL}/system/info" | grep -q "version\|name" 2>/dev/null; then
        log_success "System info endpoint"
        ((passed++))
    else
        log_error "System info endpoint failed"
        ((failed++))
    fi

    # Auth - login
    local token
    token=$(curl -s -X POST "${API_URL}/auth/login" \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
        -d '{"username":"admin","password":"test123"}' | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$token" ]; then
        log_success "Auth login endpoint"
        ((passed++))
    else
        log_error "Auth login endpoint failed"
        ((failed++))
    fi

    # Products endpoint (with auth)
    if [ -n "$token" ]; then
        if curl -s "${API_URL}/catalog/products" \
            -H "Authorization: Bearer $token" \
            -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | grep -q "data\|products" 2>/dev/null; then
            log_success "Products endpoint"
            ((passed++))
        else
            log_error "Products endpoint failed"
            ((failed++))
        fi
    fi

    echo ""
    echo -e "Quick checks: ${GREEN}$passed passed${NC}, ${RED}$failed failed${NC}"

    if [ $failed -gt 0 ]; then
        return 1
    fi
}

cmd_clean() {
    log_step "Cleaning Test Environment"

    log_info "Stopping services..."
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v

    log_info "Removing volumes..."
    docker volume rm erp-test-postgres-data erp-test-redis-data 2>/dev/null || true

    log_success "Test environment cleaned"
}

cmd_help() {
    echo "ERP Quick Test Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  start   - Start test environment and run validations (default)"
    echo "  stop    - Stop test environment"
    echo "  restart - Restart test environment"
    echo "  status  - Show status of services"
    echo "  logs    - Show logs from all services"
    echo "  seed    - Run seed data only"
    echo "  api     - Run full API smoke tests"
    echo "  clean   - Stop and remove all test data"
    echo "  help    - Show this help message"
}

# =============================================================================
# Main
# =============================================================================

main() {
    local command="${1:-start}"

    case "$command" in
        start)
            cmd_start
            ;;
        stop)
            cmd_stop
            ;;
        restart)
            cmd_restart
            ;;
        status)
            cmd_status
            ;;
        logs)
            cmd_logs "$@"
            ;;
        seed)
            cmd_seed
            ;;
        api)
            cmd_api
            ;;
        clean)
            cmd_clean
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            log_error "Unknown command: $command"
            cmd_help
            exit 1
            ;;
    esac
}

main "$@"
