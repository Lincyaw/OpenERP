-- Migration: add_tenant_billing_suspension_columns (down)
-- Description: Remove billing and suspension columns from tenants table

-- Remove indexes
DROP INDEX IF EXISTS idx_tenants_scheduled_reactivate_at;
DROP INDEX IF EXISTS idx_tenants_stripe_subscription_id;
DROP INDEX IF EXISTS idx_tenants_stripe_customer_id;

-- Remove columns
ALTER TABLE tenants DROP COLUMN IF EXISTS scheduled_reactivate_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS suspended_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS suspension_reason;
ALTER TABLE tenants DROP COLUMN IF EXISTS stripe_subscription_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS stripe_customer_id;
