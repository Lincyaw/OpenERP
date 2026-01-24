#!/bin/bash
set -e

# This script runs on first database initialization
# It creates extensions and sets up the database for the ERP system

echo "PostgreSQL initialization script started"

# Enable required extensions
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Enable UUID extension (required for uuid_generate_v4)
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

    -- Enable pgcrypto for password hashing
    CREATE EXTENSION IF NOT EXISTS "pgcrypto";
EOSQL

echo "PostgreSQL extensions created"

# Note: Migrations are run by the migrate container
# Note: Seed data can be loaded separately using docker/seed-data.sql

echo "PostgreSQL initialization completed"
