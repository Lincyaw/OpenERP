-- Migration: Create feature flags tables
-- Description: Implements feature flag system with overrides and audit logging

-- Create feature_flags table (aggregate root for feature flags)
CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('boolean', 'percentage', 'variant', 'user_segment')),
    status VARCHAR(20) NOT NULL DEFAULT 'disabled' CHECK (status IN ('enabled', 'disabled', 'archived')),
    default_value JSONB NOT NULL,
    rules JSONB,
    tags JSONB NOT NULL DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,

    -- Ensure unique key globally
    CONSTRAINT uq_feature_flag_key UNIQUE (key),

    -- Foreign keys
    CONSTRAINT fk_feature_flag_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_feature_flag_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for feature_flags
CREATE INDEX idx_feature_flags_status ON feature_flags(status);
CREATE INDEX idx_feature_flags_tags ON feature_flags USING GIN(tags);
CREATE INDEX idx_feature_flags_type ON feature_flags(type);
CREATE INDEX idx_feature_flags_created_at ON feature_flags(created_at);

-- Add update trigger for updated_at
CREATE TRIGGER trg_feature_flags_updated_at
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for feature_flags
COMMENT ON TABLE feature_flags IS 'Feature flag definitions for controlling feature rollouts';
COMMENT ON COLUMN feature_flags.key IS 'Unique identifier for the flag (e.g., new_checkout_flow)';
COMMENT ON COLUMN feature_flags.name IS 'Human-readable name for display in admin UI';
COMMENT ON COLUMN feature_flags.description IS 'Detailed description of the flag purpose and behavior';
COMMENT ON COLUMN feature_flags.type IS 'Flag type: boolean, percentage, variant, or user_segment';
COMMENT ON COLUMN feature_flags.status IS 'Flag status: enabled, disabled, or archived';
COMMENT ON COLUMN feature_flags.default_value IS 'Default value and metadata in JSON format';
COMMENT ON COLUMN feature_flags.rules IS 'Targeting rules array in JSON format';
COMMENT ON COLUMN feature_flags.tags IS 'Searchable tags for organizing flags (JSONB array)';
COMMENT ON COLUMN feature_flags.version IS 'Optimistic locking version number';

-- Create flag_overrides table for user/tenant-specific overrides
CREATE TABLE flag_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flag_key VARCHAR(100) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('user', 'tenant')),
    target_id UUID NOT NULL,
    value JSONB NOT NULL,
    reason VARCHAR(500),
    expires_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique override per flag, target type, and target id
    CONSTRAINT uq_flag_override UNIQUE (flag_key, target_type, target_id),

    -- Foreign keys
    CONSTRAINT fk_flag_override_flag_key FOREIGN KEY (flag_key) REFERENCES feature_flags(key) ON DELETE CASCADE,
    CONSTRAINT fk_flag_override_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for flag_overrides
CREATE INDEX idx_flag_overrides_target ON flag_overrides(target_type, target_id);
CREATE INDEX idx_flag_overrides_expires ON flag_overrides(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_flag_overrides_flag_key ON flag_overrides(flag_key);

-- Add comments for flag_overrides
COMMENT ON TABLE flag_overrides IS 'User or tenant-specific overrides for feature flags';
COMMENT ON COLUMN flag_overrides.flag_key IS 'Reference to the feature flag key';
COMMENT ON COLUMN flag_overrides.target_type IS 'Type of target: user or tenant';
COMMENT ON COLUMN flag_overrides.target_id IS 'ID of the user or tenant being overridden';
COMMENT ON COLUMN flag_overrides.value IS 'Override value in JSON format';
COMMENT ON COLUMN flag_overrides.reason IS 'Reason for the override';
COMMENT ON COLUMN flag_overrides.expires_at IS 'Automatic expiration timestamp (null = no expiry)';

-- Add update trigger for flag_overrides
CREATE TRIGGER trg_flag_overrides_updated_at
    BEFORE UPDATE ON flag_overrides
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create flag_audit_logs table for tracking changes
CREATE TABLE flag_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flag_key VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('created', 'updated', 'enabled', 'disabled', 'archived', 'override_added', 'override_removed')),
    old_value JSONB,
    new_value JSONB,
    user_id UUID,
    tenant_id UUID,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for flag_audit_logs
CREATE INDEX idx_flag_audit_logs_flag_key ON flag_audit_logs(flag_key);
CREATE INDEX idx_flag_audit_logs_created_at ON flag_audit_logs(created_at);
CREATE INDEX idx_flag_audit_logs_user_id ON flag_audit_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_flag_audit_logs_action ON flag_audit_logs(action);

-- Add comments for flag_audit_logs
COMMENT ON TABLE flag_audit_logs IS 'Audit log for all feature flag changes';
COMMENT ON COLUMN flag_audit_logs.flag_key IS 'The feature flag key that was modified';
COMMENT ON COLUMN flag_audit_logs.action IS 'Type of action performed on the flag';
COMMENT ON COLUMN flag_audit_logs.old_value IS 'Previous value before the change';
COMMENT ON COLUMN flag_audit_logs.new_value IS 'New value after the change';
COMMENT ON COLUMN flag_audit_logs.user_id IS 'User who performed the action';
COMMENT ON COLUMN flag_audit_logs.tenant_id IS 'Tenant context if applicable';
COMMENT ON COLUMN flag_audit_logs.ip_address IS 'IP address of the user';
COMMENT ON COLUMN flag_audit_logs.user_agent IS 'User agent string of the client';
