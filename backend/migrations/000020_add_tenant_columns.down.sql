-- Migration: add_tenant_columns (rollback)
-- Created: 2026-01-25
-- Description: Rollback tenant columns migration

-- Drop indexes first
DROP INDEX IF EXISTS idx_tenants_plan;
DROP INDEX IF EXISTS idx_tenants_trial_ends_at;
DROP INDEX IF EXISTS idx_tenants_expires_at;
DROP INDEX IF EXISTS idx_tenants_domain;

-- Drop constraints
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenant_plan;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenant_status;

-- Restore original status constraint
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_status
    CHECK (status IN ('active', 'inactive', 'suspended'));

-- Drop config embedded columns
ALTER TABLE tenants DROP COLUMN IF EXISTS config_locale;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_timezone;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_currency;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_cost_strategy;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_settings;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_features;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_max_products;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_max_warehouses;
ALTER TABLE tenants DROP COLUMN IF EXISTS config_max_users;

-- Drop main columns
ALTER TABLE tenants DROP COLUMN IF EXISTS notes;
ALTER TABLE tenants DROP COLUMN IF EXISTS trial_ends_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS expires_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS domain;
ALTER TABLE tenants DROP COLUMN IF EXISTS logo_url;
ALTER TABLE tenants DROP COLUMN IF EXISTS address;
ALTER TABLE tenants DROP COLUMN IF EXISTS contact_email;
ALTER TABLE tenants DROP COLUMN IF EXISTS contact_phone;
ALTER TABLE tenants DROP COLUMN IF EXISTS contact_name;
ALTER TABLE tenants DROP COLUMN IF EXISTS plan;
ALTER TABLE tenants DROP COLUMN IF EXISTS short_name;
