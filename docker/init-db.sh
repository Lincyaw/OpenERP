#!/bin/bash
set -e

# This script runs on first database initialization
# It can be used to create additional databases, users, or run setup scripts

echo "PostgreSQL initialization script started"

# Create additional databases if needed
# psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
#     CREATE DATABASE IF NOT EXISTS other_db;
#     GRANT ALL PRIVILEGES ON DATABASE other_db TO $POSTGRES_USER;
# EOSQL

echo "PostgreSQL initialization completed"
