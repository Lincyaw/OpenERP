# Docker Test Environment

This directory contains Docker configuration and scripts for the ERP test environment.

## Quick Start

```bash
# Start the test environment
./docker/quick-test.sh start

# Or using docker-compose directly
docker-compose -f docker-compose.test.yml up -d
```

## Files

| File | Description |
|------|-------------|
| `init-db.sh` | PostgreSQL initialization script (extensions setup) |
| `seed-data.sql` | Sample test data for all modules |
| `test-api.sh` | Comprehensive API smoke test script |
| `quick-test.sh` | Quick validation and management script |

## Test Environment Ports

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3001 | React application |
| Backend | 8081 | Go API server |
| PostgreSQL | 5433 | Database |
| Redis | 6380 | Cache |

## Commands

### Quick Test Script

```bash
# Start environment (default)
./docker/quick-test.sh start

# Stop environment
./docker/quick-test.sh stop

# Restart environment
./docker/quick-test.sh restart

# View service status
./docker/quick-test.sh status

# View logs
./docker/quick-test.sh logs

# Seed data only (requires running services)
./docker/quick-test.sh seed

# Run full API smoke tests
./docker/quick-test.sh api

# Clean up (stop and remove volumes)
./docker/quick-test.sh clean
```

### API Smoke Test Script

```bash
# Run against default URL (http://localhost:8081/api/v1)
./docker/test-api.sh

# Run against custom URL
./docker/test-api.sh http://localhost:8080/api/v1
```

## Test Credentials

| Username | Password | Role |
|----------|----------|------|
| admin | test123 | System Administrator |
| sales | test123 | Sales Manager |
| warehouse | test123 | Warehouse Manager |
| finance | test123 | Finance Manager |

## Seed Data Summary

The `seed-data.sql` file includes test data for:

- **Tenants**: 3 tenants (default + 2 test companies)
- **Categories**: 9 categories (4 root + 5 sub-categories)
- **Products**: 10 products with various prices
- **Product Units**: 3 units for A4 paper (multi-unit demo)
- **Customers**: 5 customers (3 organizations, 2 individuals)
- **Suppliers**: 5 suppliers
- **Warehouses**: 4 warehouses (3 physical, 1 virtual)
- **Inventory Items**: 10 inventory items across warehouses
- **Stock Batches**: 4 batches
- **Stock Locks**: 2 locks
- **Inventory Transactions**: 4 transactions
- **Account Receivables**: 4 receivables
- **Account Payables**: 3 payables
- **Receipt Vouchers**: 3 receipts
- **Payment Vouchers**: 2 payments
- **Expense Records**: 4 expenses
- **Other Income Records**: 2 income records
- **Balance Transactions**: 5 balance changes

## Troubleshooting

### Services won't start

1. Check if ports are already in use:
   ```bash
   lsof -i :3001 -i :8081 -i :5433 -i :6380
   ```

2. Clean up and restart:
   ```bash
   ./docker/quick-test.sh clean
   ./docker/quick-test.sh start
   ```

### Migrations fail

1. Check migration logs:
   ```bash
   docker logs erp-test-migrate
   ```

2. Reset database:
   ```bash
   docker volume rm erp-test-postgres-data
   ./docker/quick-test.sh start
   ```

### Backend not responding

1. Check backend logs:
   ```bash
   docker logs erp-test-backend
   ```

2. Verify health check:
   ```bash
   curl http://localhost:8081/health
   ```

### Seed data errors

1. Run seed data manually:
   ```bash
   PGPASSWORD=test123 psql -h localhost -p 5433 -U postgres -d erp_test -f docker/seed-data.sql
   ```

2. Check for conflicts (data may already exist):
   - Seed uses `ON CONFLICT DO NOTHING` to skip existing records

## Notes

- Test environment uses different ports to avoid conflicts with development
- All data is stored in Docker volumes (`erp-test-postgres-data`, `erp-test-redis-data`)
- Use `clean` command to completely reset the environment
- JWT secret in test environment is NOT suitable for production
