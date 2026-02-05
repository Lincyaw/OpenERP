-- Rollback migration: add_subscription_history
-- Task: P3-ADMIN-006

-- Drop subscription_history table
DROP TABLE IF EXISTS subscription_history;

-- Remove scheduled plan constraint and columns from tenants
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenant_scheduled_plan;
ALTER TABLE tenants DROP COLUMN IF EXISTS scheduled_plan;
ALTER TABLE tenants DROP COLUMN IF EXISTS scheduled_plan_effective_at;
