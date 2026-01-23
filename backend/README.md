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
│   └── server/                 # HTTP server
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
│   │   └── eventbus/          # Event bus implementation
│   └── interfaces/            # Interface layer
│       └── http/              # HTTP handlers, DTOs, middleware
├── migrations/                 # Database migrations
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

# Test
go test ./...
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
