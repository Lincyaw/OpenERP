# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DDD-based inventory management system with multi-tenancy support. Modular monolith architecture.

- **Design spec**: `.claude/ralph/docs/spec.md`
- **Frontend design system**: `frontend/README.md` (MUST READ before frontend work)

## Quick Start

```bash
make setup                # First-time setup (creates .env, installs deps)
make dev                  # Start database (postgres + redis)
make dev-backend          # Terminal 1: Run backend
make dev-frontend         # Terminal 2: Run frontend
# Access: http://localhost:3000 | Login: admin / admin123
```

## Development Commands

### Backend (Go)

```bash
# Run
cd backend && go run cmd/server/main.go

# Test
go test ./...                              # All tests
go test -v ./internal/domain/inventory/... # Single package
go test -run TestProductCreate ./...       # Single test by name
go test -cover ./...                       # With coverage
go test -race ./...                        # Race detection

# Makefile shortcuts
make test                  # All tests
make test-unit             # Unit tests only
make test-coverage         # Generate coverage report

# Migrations
./bin/migrate up           # Apply migrations
./bin/migrate down         # Rollback all
./bin/migrate create <name> "description"  # Create new migration

# API docs regeneration
make api-docs              # Generates backend/docs/swagger.yaml
```

### Frontend (React/TypeScript)

```bash
cd frontend
npm run dev                # Start dev server
npm run build              # Production build
npm run lint               # ESLint
npm run lint:fix           # Fix lint issues
npm run type-check         # TypeScript check
npm run api:generate       # Regenerate API client from OpenAPI

# Unit tests
npm run test               # Watch mode
npm run test:run           # Single run
npm run test:coverage      # With coverage
```

### E2E Testing (Playwright)

```bash
make e2e                   # Full E2E run (resets DB, runs all)
make e2e ARGS="tests/e2e/auth/auth.spec.ts"  # Single file
make e2e ARGS="--project=chromium"           # Single browser
make e2e-ui                # Playwright UI mode
make e2e-debug             # Debug mode
make e2e-local             # Against locally running services

# Database for E2E
make db-reset              # Clean + migrate + seed
make db-seed               # Load seed data only
make db-psql               # Open psql shell
```

### Observability

```bash
make otel-up               # Start OpenTelemetry Collector
make otel-status           # Check OTEL health
make pyroscope-up          # Start Pyroscope profiler
make pyroscope-ui          # Open Pyroscope UI
```

### Load Generator

```bash
make loadgen-build         # Build loadgen binary
make loadgen-test          # Run loadgen tests
```

## Architecture

### Backend DDD Layers (`backend/internal/`)

```
domain/          # Business logic (entities, value objects, domain services)
  ├── catalog/   # Product/Category management
  ├── partner/   # Customer/Supplier management
  ├── inventory/ # Stock tracking, cost calculation
  ├── trade/     # Sales/Purchase orders
  ├── finance/   # Receivables, payables, reconciliation
  ├── identity/  # User/tenant management
  └── shared/    # Shared kernel (Money, events, base types)

application/     # Use cases, orchestrates domain objects
infrastructure/  # External concerns (DB, cache, events, telemetry)
  ├── persistence/  # GORM repositories
  ├── event/        # Domain event bus
  ├── strategy/     # Strategy pattern implementations
  └── telemetry/    # OpenTelemetry, Pyroscope

interfaces/http/ # HTTP layer
  ├── handler/   # Gin handlers
  ├── dto/       # Request/response DTOs
  ├── middleware/# Auth, tenant, rate limiting
  └── router/    # Route definitions
```

### Frontend Structure (`frontend/src/`)

```
api/         # Auto-generated API client (DO NOT EDIT)
components/  # Reusable UI components
  └── common/layout/  # Container, Grid, Flex, Stack, Row
pages/       # Route pages by feature
store/       # Zustand state management
hooks/       # Custom React hooks
styles/tokens/  # Design tokens (spacing, colors, typography)
```

## API Development Workflow

1. **Backend**: Add swag annotations to handler
2. **Generate**: `make api-docs`
3. **Frontend**: `cd frontend && npm run api:generate`
4. Files in `frontend/src/api/` are auto-generated - never edit manually

## Key Patterns

### Multi-tenancy

All entities include `TenantID`. Middleware extracts tenant from JWT and injects into context.

### Domain Events

Contexts communicate via events (e.g., `OrderConfirmed` → inventory lock). Events stored in outbox table for reliability.

### Strategy Pattern

Business logic variants (cost calculation, product validation) use strategy interfaces in `infrastructure/strategy/`.

## Frontend Guidelines

- **Design tokens**: Use CSS variables (`--spacing-4`, `--color-primary`)
- **Responsive**: Mobile-first (375px → 768px → 1024px → 1440px)
- **UI library**: Semi Design (`@douyinfe/semi-ui-19`)
- **State**: Zustand for global, React Hook Form for forms
- **Accessibility**: WCAG 2.1 AA (4.5:1 contrast, 44px touch targets)

## Testing Requirements

- **Coverage threshold**: 80%
- **E2E**: Real database, no mocking
- **Backend**: testify assertions, sqlmock for DB
- **Frontend**: Vitest + React Testing Library

## Environment Configuration

Override `backend/config.toml` with `ERP_` prefix:
```bash
ERP_DATABASE_PASSWORD=secret
ERP_JWT_SECRET=my-secret
ERP_LOG_LEVEL=debug
```

## Ports

| Service | Port |
|---------|------|
| Frontend | 3000 |
| Backend | 8080 |
| PostgreSQL | 5432 |
| Redis | 6379 |
