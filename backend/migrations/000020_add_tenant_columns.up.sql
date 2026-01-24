-- Migration: add_tenant_columns
-- Created: 2026-01-25
-- Description: Add missing columns to tenants table to match domain model

-- Add new columns to tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS short_name VARCHAR(100);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan VARCHAR(20) NOT NULL DEFAULT 'free';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS contact_name VARCHAR(100);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS contact_email VARCHAR(200);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS logo_url VARCHAR(500);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS domain VARCHAR(200);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS notes TEXT;

-- Config embedded fields
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_max_users INTEGER NOT NULL DEFAULT 5;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_max_warehouses INTEGER NOT NULL DEFAULT 3;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_max_products INTEGER NOT NULL DEFAULT 1000;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_features TEXT DEFAULT '{}';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_settings TEXT DEFAULT '{}';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_cost_strategy VARCHAR(50) DEFAULT 'weighted_average';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_currency VARCHAR(10) DEFAULT 'CNY';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_timezone VARCHAR(50) DEFAULT 'Asia/Shanghai';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS config_locale VARCHAR(20) DEFAULT 'zh-CN';

-- Update status constraint to include new status values
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenant_status;
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_status
    CHECK (status IN ('active', 'inactive', 'suspended', 'trial'));

-- Add plan constraint
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_plan
    CHECK (plan IN ('free', 'basic', 'pro', 'enterprise'));

-- Create unique index on domain (nullable)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_domain ON tenants(domain) WHERE domain IS NOT NULL AND domain != '';

-- Create index on expires_at for subscription expiry queries
CREATE INDEX IF NOT EXISTS idx_tenants_expires_at ON tenants(expires_at);

-- Create index on trial_ends_at for trial expiry queries
CREATE INDEX IF NOT EXISTS idx_tenants_trial_ends_at ON tenants(trial_ends_at);

-- Create index on plan for filtering
CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan);

-- Add comments
COMMENT ON COLUMN tenants.short_name IS 'Abbreviated name for display';
COMMENT ON COLUMN tenants.plan IS 'Subscription plan (free, basic, pro, enterprise)';
COMMENT ON COLUMN tenants.contact_name IS 'Primary contact person name';
COMMENT ON COLUMN tenants.contact_phone IS 'Primary contact phone number';
COMMENT ON COLUMN tenants.contact_email IS 'Primary contact email address';
COMMENT ON COLUMN tenants.address IS 'Business address';
COMMENT ON COLUMN tenants.logo_url IS 'URL to tenant logo image';
COMMENT ON COLUMN tenants.domain IS 'Custom subdomain for tenant';
COMMENT ON COLUMN tenants.expires_at IS 'Subscription expiration date';
COMMENT ON COLUMN tenants.trial_ends_at IS 'Trial period end date';
COMMENT ON COLUMN tenants.notes IS 'Admin notes about the tenant';
COMMENT ON COLUMN tenants.config_max_users IS 'Maximum number of users allowed';
COMMENT ON COLUMN tenants.config_max_warehouses IS 'Maximum number of warehouses allowed';
COMMENT ON COLUMN tenants.config_max_products IS 'Maximum number of products allowed';
COMMENT ON COLUMN tenants.config_cost_strategy IS 'Default cost calculation strategy (fifo, weighted_average)';
COMMENT ON COLUMN tenants.config_currency IS 'Default currency code (e.g., CNY, USD)';
COMMENT ON COLUMN tenants.config_timezone IS 'Tenant timezone (e.g., Asia/Shanghai)';
COMMENT ON COLUMN tenants.config_locale IS 'Tenant locale (e.g., zh-CN, en-US)';
