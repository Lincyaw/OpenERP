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
│   │   └── eventbus/          # Event bus implementation
│   └── interfaces/            # Interface layer
│       └── http/              # HTTP handlers, DTOs, middleware
├── migrations/                 # Database migrations (SQL files)
└── tests/                     # Tests
    ├── unit/
    └── integration/
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
