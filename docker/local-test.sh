#!/bin/bash
# =============================================================================
# ERP Local Test Environment Script
# =============================================================================
# This script manages a local E2E test environment where:
#   - PostgreSQL and Redis run in Docker
#   - Backend and Frontend run locally (with logs captured to logs/ directory)
#   - Suitable for local debugging and agent E2E testing
#
# Usage:
#   ./docker/local-test.sh [command]
#
# Commands:
#   start       - Start everything (database + backend + frontend)
#   start-db    - Start database services only
#   start-all   - Start backend and frontend (assumes database is running)
#   stop        - Stop everything
#   stop-db     - Stop database services only
#   stop-all    - Stop backend and frontend processes
#   restart     - Restart everything
#   status      - Show status of services
#   logs        - Show logs from database services
#   logs-backend  - Tail backend logs
#   logs-frontend - Tail frontend logs
#   seed        - Load seed data into database
#   migrate     - Run database migrations
#   clean       - Stop and remove all data
#   run-e2e     - Run E2E tests
#   help        - Show this help message
#
# Exit codes:
#   0 - Success
#   1 - Failure
# =============================================================================

set -e

# Configuration
COMPOSE_FILE="docker/docker-compose.local-db.yml"
PROJECT_NAME="erp-local"
DB_HOST="localhost"
DB_PORT="5433"
DB_NAME="erp_test"
DB_USER="postgres"
DB_PASSWORD="test123"
BACKEND_PORT="8081"
FRONTEND_PORT="3001"
MIGRATE_IMAGE="migrate/migrate:v4.17.0"
LOGS_DIR="logs"
BACKEND_LOG="$LOGS_DIR/backend.log"
FRONTEND_LOG="$LOGS_DIR/frontend.log"
BACKEND_PID_FILE="$LOGS_DIR/backend.pid"
FRONTEND_PID_FILE="$LOGS_DIR/frontend.pid"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Working directory - ensure we're at project root
cd "$(dirname "$0")/.."

# Create logs directory
mkdir -p "$LOGS_DIR"

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

log_cyan() {
    echo -e "${CYAN}$1${NC}"
}

# Check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is required but not installed."
        return 1
    fi
}

# Wait for database to be ready
wait_for_database() {
    local max_attempts="${1:-30}"
    local attempt=1

    log_info "Waiting for database connection..."

    while [ $attempt -le $max_attempts ]; do
        # Try local psql first, then fallback to docker exec
        if command -v psql &> /dev/null; then
            if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &>/dev/null; then
                log_success "Database is ready"
                return 0
            fi
        else
            if docker exec erp-local-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" &>/dev/null; then
                log_success "Database is ready"
                return 0
            fi
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done

    echo ""
    log_error "Database connection failed after $max_attempts attempts"
    return 1
}

# Wait for Redis to be ready
wait_for_redis() {
    local max_attempts="${1:-30}"
    local attempt=1

    log_info "Waiting for Redis connection..."

    while [ $attempt -le $max_attempts ]; do
        if redis-cli -h "$DB_HOST" -p 6380 ping &>/dev/null; then
            log_success "Redis is ready"
            return 0
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done

    echo ""
    log_error "Redis connection failed after $max_attempts attempts"
    return 1
}

# Check if backend is running
check_backend() {
    if curl -s "http://localhost:$BACKEND_PORT/health" &>/dev/null; then
        return 0
    fi
    return 1
}

# Check if frontend is running
check_frontend() {
    if curl -s "http://localhost:$FRONTEND_PORT" &>/dev/null; then
        return 0
    fi
    return 1
}

# Get process ID from PID file
get_pid() {
    local pid_file="$1"
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo "$pid"
            return 0
        fi
    fi
    echo ""
    return 1
}

# Stop a process by PID file
stop_process() {
    local pid_file="$1"
    local name="$2"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            log_info "Stopping $name (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            sleep 2
            # Force kill if still running
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
            fi
            log_success "$name stopped"
        fi
        rm -f "$pid_file"
    fi
}

# =============================================================================
# Commands
# =============================================================================

cmd_start_db() {
    log_step "Starting Local Test Database Environment"

    # Check prerequisites
    check_command "docker" || exit 1

    # Start services
    log_info "Starting PostgreSQL and Redis in Docker..."
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d

    # Wait for database to be ready
    log_step "Waiting for Services"
    sleep 5
    wait_for_database 30 || exit 1

    # Run migrations
    cmd_migrate

    # Seed data
    cmd_seed

    log_success "Database services are ready"
}

cmd_start_backend() {
    log_step "Starting Backend"

    # Check if already running
    if check_backend; then
        log_warning "Backend is already running on localhost:$BACKEND_PORT"
        return 0
    fi

    # Copy env file if not exists
    if [ ! -f "backend/.env" ] || ! grep -q "APP_PORT=$BACKEND_PORT" "backend/.env" 2>/dev/null; then
        log_info "Setting up backend environment..."
        cp backend/.env.local-test backend/.env
    fi

    log_info "Starting backend server (logs: $BACKEND_LOG)..."

    cd backend
    # Build backend
    go build -o ../bin/erp-server ./cmd/server || { cd ..; return 1; }
    cd ..

    # Export environment variables from .env file and run backend in background
    # Use env to set variables for the command
    env $(grep -v '^#' backend/.env | xargs) nohup ./bin/erp-server > "$BACKEND_LOG" 2>&1 &
    local pid=$!
    echo $pid > "$BACKEND_PID_FILE"

    # Wait for backend to be ready
    local max_attempts=30
    local attempt=1
    while [ $attempt -le $max_attempts ]; do
        if check_backend; then
            log_success "Backend is ready (PID: $pid)"
            return 0
        fi
        echo -n "."
        sleep 1
        ((attempt++))
    done

    echo ""
    log_error "Backend failed to start. Check logs: $BACKEND_LOG"
    return 1
}

cmd_start_frontend() {
    log_step "Starting Frontend"

    # Check if already running
    if check_frontend; then
        log_warning "Frontend is already running on localhost:$FRONTEND_PORT"
        return 0
    fi

    log_info "Starting frontend dev server (logs: $FRONTEND_LOG)..."

    cd frontend
    # Start frontend in background with proper environment
    VITE_API_BASE_URL="http://localhost:$BACKEND_PORT/api/v1" \
    nohup npm run dev -- --port $FRONTEND_PORT > "../$FRONTEND_LOG" 2>&1 &
    local pid=$!
    echo $pid > "../$FRONTEND_PID_FILE"
    cd ..

    # Wait for frontend to be ready
    local max_attempts=60
    local attempt=1
    while [ $attempt -le $max_attempts ]; do
        if check_frontend; then
            log_success "Frontend is ready (PID: $pid)"
            return 0
        fi
        echo -n "."
        sleep 1
        ((attempt++))
    done

    echo ""
    log_error "Frontend failed to start. Check logs: $FRONTEND_LOG"
    return 1
}

cmd_start_all() {
    cmd_start_backend
    cmd_start_frontend
}

cmd_start() {
    log_step "Starting Complete Local Test Environment"

    # Start database first
    cmd_start_db

    # Start backend and frontend
    cmd_start_all

    log_step "Local Test Environment Ready!"
    echo ""
    echo -e "  ${GREEN}PostgreSQL:${NC}  localhost:5433"
    echo -e "  ${GREEN}Redis:${NC}       localhost:6380"
    echo -e "  ${GREEN}Backend:${NC}     http://localhost:$BACKEND_PORT"
    echo -e "  ${GREEN}Frontend:${NC}    http://localhost:$FRONTEND_PORT"
    echo ""
    echo -e "  ${CYAN}Logs:${NC}"
    echo "    Backend:  $BACKEND_LOG"
    echo "    Frontend: $FRONTEND_LOG"
    echo ""
    echo -e "  ${BLUE}Login credentials:${NC}"
    echo "    Username: admin"
    echo "    Password: test123"
    echo ""
    echo -e "  ${CYAN}Run E2E tests:${NC}"
    echo "    ./docker/local-test.sh run-e2e"
    echo ""
}

cmd_stop_db() {
    log_step "Stopping Database Services"
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down 2>/dev/null || true
    log_success "Database services stopped"
}

cmd_stop_all() {
    log_step "Stopping Backend and Frontend"
    stop_process "$BACKEND_PID_FILE" "Backend"
    stop_process "$FRONTEND_PID_FILE" "Frontend"
}

cmd_stop() {
    cmd_stop_all
    cmd_stop_db
    log_success "All services stopped"
}

cmd_restart() {
    cmd_stop
    cmd_start
}

cmd_status() {
    log_step "Service Status"

    echo ""
    echo "Docker Services:"
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps 2>/dev/null || echo "  No Docker services running"

    echo ""
    echo "Local Services:"

    local backend_pid=$(get_pid "$BACKEND_PID_FILE")
    if [ -n "$backend_pid" ] && check_backend; then
        echo -e "  Backend (localhost:$BACKEND_PORT):   ${GREEN}Running${NC} (PID: $backend_pid)"
    elif check_backend; then
        echo -e "  Backend (localhost:$BACKEND_PORT):   ${GREEN}Running${NC} (external process)"
    else
        echo -e "  Backend (localhost:$BACKEND_PORT):   ${RED}Not Running${NC}"
    fi

    local frontend_pid=$(get_pid "$FRONTEND_PID_FILE")
    if [ -n "$frontend_pid" ] && check_frontend; then
        echo -e "  Frontend (localhost:$FRONTEND_PORT):  ${GREEN}Running${NC} (PID: $frontend_pid)"
    elif check_frontend; then
        echo -e "  Frontend (localhost:$FRONTEND_PORT):  ${GREEN}Running${NC} (external process)"
    else
        echo -e "  Frontend (localhost:$FRONTEND_PORT):  ${RED}Not Running${NC}"
    fi

    echo ""
    echo "Log Files:"
    if [ -f "$BACKEND_LOG" ]; then
        echo -e "  Backend:  ${GREEN}$BACKEND_LOG${NC} ($(wc -l < "$BACKEND_LOG") lines)"
    else
        echo -e "  Backend:  ${YELLOW}Not created yet${NC}"
    fi
    if [ -f "$FRONTEND_LOG" ]; then
        echo -e "  Frontend: ${GREEN}$FRONTEND_LOG${NC} ($(wc -l < "$FRONTEND_LOG") lines)"
    else
        echo -e "  Frontend: ${YELLOW}Not created yet${NC}"
    fi
    echo ""
}

cmd_logs_db() {
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f "${@:2}"
}

cmd_logs_backend() {
    if [ -f "$BACKEND_LOG" ]; then
        tail -f "$BACKEND_LOG"
    else
        log_error "Backend log file not found: $BACKEND_LOG"
        exit 1
    fi
}

cmd_logs_frontend() {
    if [ -f "$FRONTEND_LOG" ]; then
        tail -f "$FRONTEND_LOG"
    else
        log_error "Frontend log file not found: $FRONTEND_LOG"
        exit 1
    fi
}

cmd_migrate() {
    log_step "Running Database Migrations"

    log_info "Running migrations via Docker..."
    docker run --rm \
        --network host \
        -v "$(pwd)/backend/migrations:/migrations:ro" \
        "$MIGRATE_IMAGE" \
        -path=/migrations \
        -database "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" \
        up 2>/dev/null || log_warning "Migrations may already be applied"

    log_success "Migrations completed"
}

cmd_seed() {
    log_step "Seeding Test Data"

    # Check if psql is available
    if command -v psql &> /dev/null; then
        log_info "Seeding data via local psql..."
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f docker/seed-data.sql 2>/dev/null && \
            log_success "Seed data applied" || \
            log_warning "Seed data may have already been applied (conflicts ignored)"
    else
        log_info "Seeding data via Docker..."
        docker exec -i erp-local-postgres psql -U "$DB_USER" -d "$DB_NAME" < docker/seed-data.sql 2>/dev/null && \
            log_success "Seed data applied" || \
            log_warning "Seed data may have already been applied (conflicts ignored)"
    fi
}

cmd_clean() {
    log_step "Cleaning Local Test Environment"

    # Stop all processes
    cmd_stop_all

    # Stop database
    log_info "Stopping database services..."
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v 2>/dev/null || true

    log_info "Removing volumes..."
    docker volume rm erp-local-postgres-data erp-local-redis-data 2>/dev/null || true

    # Clean log files
    log_info "Cleaning log files..."
    rm -f "$BACKEND_LOG" "$FRONTEND_LOG" "$BACKEND_PID_FILE" "$FRONTEND_PID_FILE"

    # Clean built binary
    rm -f bin/erp-server

    log_success "Local test environment cleaned"
}

cmd_run_e2e() {
    log_step "Running E2E Tests"

    # Check prerequisites
    if ! check_backend; then
        log_error "Backend is not running on localhost:$BACKEND_PORT"
        echo ""
        echo "Start the environment first: ./docker/local-test.sh start"
        exit 1
    fi

    if ! check_frontend; then
        log_error "Frontend is not running on localhost:$FRONTEND_PORT"
        echo ""
        echo "Start the environment first: ./docker/local-test.sh start"
        exit 1
    fi

    log_success "Backend and Frontend are running"
    log_info "Running Playwright E2E tests in Docker..."

    # Get absolute path to project root (current directory after cd in script)
    local project_root
    project_root="$(pwd)"

    # Run Playwright tests in Docker with current user to avoid permission issues
    # Note: node_modules is mounted from host, so dependencies are available
    docker run --rm \
        --user "$(id -u):$(id -g)" \
        --network host \
        -v "$project_root/frontend:/work" \
        -w /work \
        -e HOME=/tmp \
        -e E2E_BASE_URL="http://localhost:$FRONTEND_PORT" \
        mcr.microsoft.com/playwright:v1.58.0-noble \
        node node_modules/@playwright/test/cli.js test --config=playwright.config.ts "$@"

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        log_success "E2E tests passed!"
    else
        log_error "E2E tests failed with exit code $exit_code"
    fi

    return $exit_code
}

cmd_run_e2e_ui() {
    log_step "Running E2E Tests with UI"

    # Check prerequisites
    if ! check_backend; then
        log_error "Backend is not running on localhost:$BACKEND_PORT"
        exit 1
    fi

    if ! check_frontend; then
        log_error "Frontend is not running on localhost:$FRONTEND_PORT"
        exit 1
    fi

    # UI mode requires local playwright (not Docker) for interactive UI
    cd frontend
    E2E_BASE_URL="http://localhost:$FRONTEND_PORT" npx playwright test --ui
    cd ..
}

cmd_run_e2e_debug() {
    log_step "Running E2E Tests in Debug Mode"

    # Check prerequisites
    if ! check_backend; then
        log_error "Backend is not running on localhost:$BACKEND_PORT"
        exit 1
    fi

    if ! check_frontend; then
        log_error "Frontend is not running on localhost:$FRONTEND_PORT"
        exit 1
    fi

    # Debug mode requires local playwright (not Docker) for interactive debugging
    cd frontend
    E2E_BASE_URL="http://localhost:$FRONTEND_PORT" npx playwright test --debug "$@"
    cd ..
}

cmd_api_check() {
    log_step "Running Quick API Checks"

    local passed=0
    local failed=0

    # Health check
    if curl -s "http://localhost:$BACKEND_PORT/health" | grep -q "ok\|healthy" 2>/dev/null; then
        log_success "Health endpoint"
        ((passed++))
    else
        log_error "Health endpoint failed"
        ((failed++))
    fi

    # System info
    if curl -s "http://localhost:$BACKEND_PORT/api/v1/system/info" | grep -q "version\|name" 2>/dev/null; then
        log_success "System info endpoint"
        ((passed++))
    else
        log_error "System info endpoint failed"
        ((failed++))
    fi

    # Auth - login
    local token
    token=$(curl -s -X POST "http://localhost:$BACKEND_PORT/api/v1/auth/login" \
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
        if curl -s "http://localhost:$BACKEND_PORT/api/v1/catalog/products" \
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

cmd_help() {
    echo "ERP Local Test Environment Script"
    echo ""
    echo "This script manages a local E2E test environment where PostgreSQL and Redis"
    echo "run in Docker, and backend/frontend run locally with logs captured."
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  start         - Start everything (database + backend + frontend)"
    echo "  start-db      - Start database services only"
    echo "  start-all     - Start backend and frontend only"
    echo "  stop          - Stop everything"
    echo "  stop-db       - Stop database services only"
    echo "  stop-all      - Stop backend and frontend only"
    echo "  restart       - Restart everything"
    echo "  status        - Show status of all services"
    echo ""
    echo "  logs          - Show database logs"
    echo "  logs-backend  - Tail backend log file"
    echo "  logs-frontend - Tail frontend log file"
    echo ""
    echo "  seed          - Load seed data into database"
    echo "  migrate       - Run database migrations"
    echo "  clean         - Stop everything and remove all data"
    echo ""
    echo "  run-e2e       - Run E2E tests (pass additional args to playwright)"
    echo "  run-e2e-ui    - Run E2E tests with Playwright UI"
    echo "  run-e2e-debug - Run E2E tests in debug mode"
    echo "  api-check     - Run quick API smoke tests"
    echo ""
    echo "  help          - Show this help message"
    echo ""
    echo "Log Files:"
    echo "  Backend:  logs/backend.log"
    echo "  Frontend: logs/frontend.log"
    echo ""
    echo "Examples:"
    echo "  $0 start                          # Start complete environment"
    echo "  $0 status                         # Check service status"
    echo "  $0 run-e2e                        # Run all E2E tests"
    echo "  $0 run-e2e --project=chromium     # Run E2E tests in Chrome only"
    echo "  $0 run-e2e tests/e2e/auth/        # Run specific test directory"
    echo "  $0 run-e2e-debug auth.spec.ts     # Debug specific test"
    echo "  $0 logs-backend                   # Watch backend logs"
    echo "  $0 stop                           # Stop everything"
    echo "  $0 clean                          # Clean up everything"
}

# =============================================================================
# Main
# =============================================================================

main() {
    local command="${1:-help}"

    case "$command" in
        start)
            cmd_start
            ;;
        start-db)
            cmd_start_db
            ;;
        start-all|start-services)
            cmd_start_all
            ;;
        stop)
            cmd_stop
            ;;
        stop-db)
            cmd_stop_db
            ;;
        stop-all|stop-services)
            cmd_stop_all
            ;;
        restart)
            cmd_restart
            ;;
        status)
            cmd_status
            ;;
        logs)
            cmd_logs_db "$@"
            ;;
        logs-backend)
            cmd_logs_backend
            ;;
        logs-frontend)
            cmd_logs_frontend
            ;;
        seed)
            cmd_seed
            ;;
        migrate)
            cmd_migrate
            ;;
        clean)
            cmd_clean
            ;;
        run-e2e)
            shift
            cmd_run_e2e "$@"
            ;;
        run-e2e-ui)
            cmd_run_e2e_ui
            ;;
        run-e2e-debug)
            shift
            cmd_run_e2e_debug "$@"
            ;;
        api-check|api)
            cmd_api_check
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
