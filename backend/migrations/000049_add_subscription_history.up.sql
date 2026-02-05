-- Migration: add_subscription_history
-- Created: 2026-02-02
-- Description: Add subscription history table and scheduled plan fields to tenants
-- Task: P3-ADMIN-006

-- Add scheduled plan fields to tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS scheduled_plan VARCHAR(20);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS scheduled_plan_effective_at TIMESTAMPTZ;

-- Add constraint for scheduled_plan
ALTER TABLE tenants ADD CONSTRAINT chk_tenant_scheduled_plan
    CHECK (scheduled_plan IS NULL OR scheduled_plan IN ('free', 'basic', 'pro', 'enterprise'));

-- Create subscription_history table
CREATE TABLE IF NOT EXISTS subscription_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    change_type VARCHAR(30) NOT NULL,
    old_plan VARCHAR(20),
    new_plan VARCHAR(20),
    old_quota JSONB,
    new_quota JSONB,
    effective_at TIMESTAMPTZ NOT NULL,
    scheduled_at TIMESTAMPTZ,
    changed_by_user_id UUID NOT NULL REFERENCES users(id),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_subscription_history_change_type
        CHECK (change_type IN ('plan_upgrade', 'plan_downgrade', 'quota_update')),
    CONSTRAINT chk_subscription_history_old_plan
        CHECK (old_plan IS NULL OR old_plan IN ('free', 'basic', 'pro', 'enterprise')),
    CONSTRAINT chk_subscription_history_new_plan
        CHECK (new_plan IS NULL OR new_plan IN ('free', 'basic', 'pro', 'enterprise'))
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_subscription_history_tenant_id
    ON subscription_history(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subscription_history_created_at
    ON subscription_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscription_history_tenant_created
    ON subscription_history(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscription_history_effective_at
    ON subscription_history(effective_at);

-- Index for finding pending downgrades
CREATE INDEX IF NOT EXISTS idx_tenants_scheduled_plan
    ON tenants(scheduled_plan_effective_at)
    WHERE scheduled_plan IS NOT NULL;

-- Comments
COMMENT ON TABLE subscription_history IS 'Tracks all subscription plan and quota changes for tenants';
COMMENT ON COLUMN subscription_history.change_type IS 'Type of change: plan_upgrade, plan_downgrade, quota_update';
COMMENT ON COLUMN subscription_history.effective_at IS 'When the change takes/took effect';
COMMENT ON COLUMN subscription_history.scheduled_at IS 'For downgrades: when the change was scheduled';
COMMENT ON COLUMN tenants.scheduled_plan IS 'Plan to change to at billing cycle end (for downgrades)';
COMMENT ON COLUMN tenants.scheduled_plan_effective_at IS 'When scheduled plan change takes effect';
