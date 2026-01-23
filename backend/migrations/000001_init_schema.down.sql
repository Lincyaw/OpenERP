-- Migration: init_schema (Rollback)
-- Created: 2026-01-23
-- Description: Rollback initial schema setup

-- Remove default tenant
DELETE FROM tenants WHERE code = 'default';

-- Drop trigger
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tenants table
DROP TABLE IF EXISTS tenants;

-- Note: We don't drop extensions as they might be used by other schemas
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
