# ERP Backend

Go backend for the ERP inventory management system using DDD architecture.

## Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+

## Project Structure

```
backend/
├── cmd/                        # Entry points
│   ├── server/                 # HTTP server
│   └── migrate/                # Database migration CLI
├── internal/
│   ├── domain/                 # Domain layer (DDD)
│   │   ├── catalog/           # Product/Category context
│   │   ├── partner/           # Customer/Supplier context
│   │   ├── inventory/         # Inventory context
│   │   ├── trade/             # Sales/Purchase order context
│   │   ├── finance/           # Financial context
│   │   └── shared/            # Shared kernel (value objects, events)
│   ├── application/           # Application services
│   ├── infrastructure/        # Infrastructure layer
│   │   ├── config/            # Configuration
│   │   ├── persistence/       # Database repositories
│   │   ├── migration/         # Migration utilities
│   │   ├── event/             # Event bus and outbox
│   │   ├── logger/            # Structured logging
│   │   └── strategy/          # Strategy registry and implementations
│   └── interfaces/            # Interface layer
│       └── http/              # HTTP handlers, DTOs, middleware
├── migrations/                 # Database migrations (SQL files)
└── tests/
    └── testutil/              # Test utilities and helpers
```

## Running

```bash
# Development
go run cmd/server/main.go

# Build
go build -o bin/server cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go

# Test
go test ./...
```

## Testing

The project uses [testify](https://github.com/stretchr/testify) for assertions and [sqlmock](https://github.com/DATA-DOG/go-sqlmock) for database mocking.

### Running Tests

```bash
# Using Makefile (recommended)
make test              # Run all tests
make test-unit         # Run unit tests only
make test-race         # Run with race detector
make test-coverage     # Run with coverage report
make test-coverage-html # Generate HTML coverage report

# Using go test directly
go test ./...                          # All tests
go test -v ./internal/...              # Verbose output
go test -cover ./...                   # With coverage
go test -race ./...                    # Race detection
go test -coverprofile=coverage.out ./... # Coverage file
go tool cover -html=coverage.out       # View HTML report
```

### Test Utilities

The `tests/testutil` package provides reusable test helpers:

```go
import "github.com/erp/backend/tests/testutil"

// Mock database
mockDB := testutil.NewMockDB(t)
defer mockDB.Close()
mockDB.Mock.ExpectQuery(...).WillReturnRows(...)

// HTTP test context
tc := testutil.NewTestContext(t)
tc.SetRequestID("req-123")
tc.SetTenantID("tenant-456")

// Deterministic UUIDs for reproducible tests
tenantID := testutil.TestTenantID()
userID := testutil.TestUserID()

// Event testing
handler := testutil.NewMockEventHandler("ProductCreated")
event := testutil.NewTestEvent("ProductCreated", tenantID)

// Async assertions
testutil.AssertEventually(t, func() bool {
    return condition
}, 5*time.Second, 100*time.Millisecond)
```

### Test Coverage

Coverage threshold is 80%. Check coverage for CI:

```bash
make test-coverage-ci  # Fails if coverage < 80%
```

### Test Patterns

- **Unit tests**: Co-located with source files (`*_test.go`)
- **Table-driven tests**: Use `t.Run()` for subtests
- **Mocking**: Use interface mocks, not concrete implementations
- **Assertions**: Use `require` for setup, `assert` for verification

## Database Migrations

The project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations.

### Migration CLI

```bash
# Build the migration CLI
go build -o bin/migrate cmd/migrate/main.go

# Apply all pending migrations
./bin/migrate up

# Roll back all migrations
./bin/migrate down

# Roll back the last migration
./bin/migrate step -1

# Apply the next 2 migrations
./bin/migrate step 2

# Migrate to a specific version
./bin/migrate goto 000001

# Check current migration version
./bin/migrate version

# Create a new migration
./bin/migrate create add_users_table "Create users table"

# List available migrations
./bin/migrate list

# Force set version (use with caution - for fixing dirty state)
./bin/migrate force 000001

# Drop all database objects (DANGEROUS)
./bin/migrate drop -confirm
```

### Migration Files

Migrations are stored in `migrations/` directory with the format:
- `{version}_{name}.up.sql` - Forward migration
- `{version}_{name}.down.sql` - Rollback migration

Example:
```
migrations/
├── 000001_init_schema.up.sql
├── 000001_init_schema.down.sql
├── 000002_add_users.up.sql
└── 000002_add_users.down.sql
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| APP_NAME | Application name | erp-backend |
| APP_ENV | Environment (development/production) | development |
| APP_PORT | HTTP server port | 8080 |
| DB_HOST | PostgreSQL host | localhost |
| DB_PORT | PostgreSQL port | 5432 |
| DB_USER | PostgreSQL user | postgres |
| DB_PASSWORD | PostgreSQL password | - |
| DB_NAME | Database name | erp |
| DB_SSL_MODE | SSL mode | disable |
| REDIS_HOST | Redis host | localhost |
| REDIS_PORT | Redis port | 6379 |
| JWT_SECRET | JWT signing secret | - |
