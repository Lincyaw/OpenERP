-- Migration: create_admin_audit_logs
-- Created: 2026-02-01
-- Description: Creates admin audit logs table for tracking super admin operations
-- Task: P3-ADMIN-003

-- ============================================================================
-- ADMIN AUDIT LOGS TABLE
-- ============================================================================
-- This table stores audit logs for all super admin operations.
-- Logs are immutable - no UPDATE or DELETE operations should be performed.

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Who performed the action
    admin_user_id UUID NOT NULL REFERENCES users(id),

    -- What action was performed
    action VARCHAR(50) NOT NULL,

    -- Target of the action
    target_type VARCHAR(50) NOT NULL,
    target_id UUID,

    -- Before/after values for audit trail (JSONB for flexibility)
    old_value JSONB,
    new_value JSONB,

    -- Request context
    ip_address VARCHAR(45),  -- IPv6 max length
    user_agent VARCHAR(500),

    -- Timestamp (immutable)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- INDEXES FOR QUERY PERFORMANCE
-- ============================================================================

-- Index for querying by admin user
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_admin_user_id
    ON admin_audit_logs(admin_user_id);

-- Index for querying by action type
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_action
    ON admin_audit_logs(action);

-- Index for querying by target
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_target
    ON admin_audit_logs(target_type, target_id);

-- Index for time-based queries (most common)
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_created_at
    ON admin_audit_logs(created_at DESC);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_action_created_at
    ON admin_audit_logs(action, created_at DESC);

-- Composite index for target + time queries
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_target_created_at
    ON admin_audit_logs(target_type, target_id, created_at DESC);

-- ============================================================================
-- CONSTRAINTS
-- ============================================================================

-- Ensure action is one of the allowed types
ALTER TABLE admin_audit_logs ADD CONSTRAINT chk_admin_audit_logs_action
    CHECK (action IN (
        'TENANT_CREATE',
        'TENANT_UPDATE',
        'TENANT_DELETE',
        'TENANT_SUSPEND',
        'TENANT_ACTIVATE',
        'SUBSCRIPTION_CHANGE',
        'QUOTA_UPDATE',
        'USER_CREATE',
        'USER_UPDATE',
        'USER_DELETE',
        'USER_SUSPEND',
        'USER_ACTIVATE',
        'ROLE_CREATE',
        'ROLE_UPDATE',
        'ROLE_DELETE',
        'PERMISSION_GRANT',
        'PERMISSION_REVOKE',
        'SYSTEM_CONFIG_CHANGE'
    ));

-- Ensure target_type is one of the allowed types
ALTER TABLE admin_audit_logs ADD CONSTRAINT chk_admin_audit_logs_target_type
    CHECK (target_type IN (
        'tenant',
        'user',
        'role',
        'permission',
        'subscription',
        'quota',
        'system_config'
    ));

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE admin_audit_logs IS 'Immutable audit log for super admin operations. DO NOT UPDATE or DELETE records.';
COMMENT ON COLUMN admin_audit_logs.id IS 'Unique identifier for the audit log entry';
COMMENT ON COLUMN admin_audit_logs.admin_user_id IS 'ID of the super admin user who performed the action';
COMMENT ON COLUMN admin_audit_logs.action IS 'Type of action performed (e.g., TENANT_CREATE, TENANT_SUSPEND)';
COMMENT ON COLUMN admin_audit_logs.target_type IS 'Type of entity affected (e.g., tenant, user, role)';
COMMENT ON COLUMN admin_audit_logs.target_id IS 'ID of the affected entity (nullable for system-wide actions)';
COMMENT ON COLUMN admin_audit_logs.old_value IS 'JSON snapshot of entity state before the action (for updates/deletes)';
COMMENT ON COLUMN admin_audit_logs.new_value IS 'JSON snapshot of entity state after the action (for creates/updates)';
COMMENT ON COLUMN admin_audit_logs.ip_address IS 'IP address of the admin user (IPv4 or IPv6)';
COMMENT ON COLUMN admin_audit_logs.user_agent IS 'User agent string from the HTTP request';
COMMENT ON COLUMN admin_audit_logs.created_at IS 'Timestamp when the action was performed';

-- ============================================================================
-- SECURITY: Prevent modifications to audit logs
-- ============================================================================
-- Note: In production, consider using row-level security or triggers to
-- prevent UPDATE and DELETE operations on this table.

-- Create a trigger to prevent updates
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable and cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_audit_log_update
    BEFORE UPDATE ON admin_audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_log_modification();

CREATE TRIGGER trg_prevent_audit_log_delete
    BEFORE DELETE ON admin_audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_log_modification();
