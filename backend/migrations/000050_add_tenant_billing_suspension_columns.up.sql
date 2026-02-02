-- Migration: add_tenant_billing_suspension_columns
-- Created: 2026-02-02
-- Description: Add billing (Stripe) and suspension columns to tenants table
-- Task: P3-ADMIN-020

-- Add Stripe billing columns
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(255);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS stripe_subscription_id VARCHAR(255);

-- Create indexes for Stripe lookups
CREATE INDEX IF NOT EXISTS idx_tenants_stripe_customer_id
    ON tenants(stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_stripe_subscription_id
    ON tenants(stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;

-- Add suspension-related columns
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS suspension_reason TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS scheduled_reactivate_at TIMESTAMPTZ;

-- Create index for scheduled reactivation queries
CREATE INDEX IF NOT EXISTS idx_tenants_scheduled_reactivate_at
    ON tenants(scheduled_reactivate_at)
    WHERE scheduled_reactivate_at IS NOT NULL;

-- Comments
COMMENT ON COLUMN tenants.stripe_customer_id IS 'Stripe customer ID for billing';
COMMENT ON COLUMN tenants.stripe_subscription_id IS 'Stripe subscription ID for billing';
COMMENT ON COLUMN tenants.suspension_reason IS 'Reason for tenant suspension (set by admin)';
COMMENT ON COLUMN tenants.suspended_at IS 'When the tenant was suspended';
COMMENT ON COLUMN tenants.scheduled_reactivate_at IS 'When the tenant is scheduled to be reactivated (for temporary suspensions)';
