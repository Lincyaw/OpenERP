-- Migration: Drop feature flags tables
-- Description: Rollback feature flag system implementation

-- Drop audit logs table
DROP TABLE IF EXISTS flag_audit_logs;

-- Drop overrides table
DROP TABLE IF EXISTS flag_overrides;

-- Drop feature flags table (will cascade to triggers)
DROP TABLE IF EXISTS feature_flags;
