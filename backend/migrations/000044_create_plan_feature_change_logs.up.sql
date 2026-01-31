-- Migration: Create plan_feature_change_logs table
-- Description: Audit log for plan feature configuration changes

-- Create plan_feature_change_logs table
CREATE TABLE plan_feature_change_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_id VARCHAR(50) NOT NULL,
    feature_key VARCHAR(100) NOT NULL,
    change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('created', 'updated', 'deleted')),
    old_enabled BOOLEAN,
    new_enabled BOOLEAN,
    old_limit INTEGER,
    new_limit INTEGER,
    changed_by UUID,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for plan_feature_change_logs
CREATE INDEX idx_plan_feature_change_logs_plan_id ON plan_feature_change_logs(plan_id);
CREATE INDEX idx_plan_feature_change_logs_feature_key ON plan_feature_change_logs(feature_key);
CREATE INDEX idx_plan_feature_change_logs_changed_at ON plan_feature_change_logs(changed_at);
CREATE INDEX idx_plan_feature_change_logs_changed_by ON plan_feature_change_logs(changed_by);

-- Add comments
COMMENT ON TABLE plan_feature_change_logs IS 'Audit log for plan feature configuration changes';
COMMENT ON COLUMN plan_feature_change_logs.plan_id IS 'The plan that was modified';
COMMENT ON COLUMN plan_feature_change_logs.feature_key IS 'The feature that was modified';
COMMENT ON COLUMN plan_feature_change_logs.change_type IS 'Type of change: created, updated, or deleted';
COMMENT ON COLUMN plan_feature_change_logs.old_enabled IS 'Previous enabled state (NULL for created)';
COMMENT ON COLUMN plan_feature_change_logs.new_enabled IS 'New enabled state (NULL for deleted)';
COMMENT ON COLUMN plan_feature_change_logs.old_limit IS 'Previous limit value (NULL for created or unlimited)';
COMMENT ON COLUMN plan_feature_change_logs.new_limit IS 'New limit value (NULL for deleted or unlimited)';
COMMENT ON COLUMN plan_feature_change_logs.changed_by IS 'User who made the change';
COMMENT ON COLUMN plan_feature_change_logs.changed_at IS 'Timestamp of the change';
