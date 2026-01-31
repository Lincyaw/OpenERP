-- Migration: Drop usage metering tables
-- Description: Removes usage records and quotas tables

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_usage_quotas_updated_at ON usage_quotas;
DROP TRIGGER IF EXISTS trg_usage_records_updated_at ON usage_records;

-- Drop indexes
DROP INDEX IF EXISTS idx_usage_quotas_plan_type_active;
DROP INDEX IF EXISTS idx_usage_quotas_is_active;
DROP INDEX IF EXISTS idx_usage_quotas_usage_type;
DROP INDEX IF EXISTS idx_usage_quotas_tenant_id;
DROP INDEX IF EXISTS idx_usage_quotas_plan_id;

DROP INDEX IF EXISTS idx_usage_records_recent;
DROP INDEX IF EXISTS idx_usage_records_user_id;
DROP INDEX IF EXISTS idx_usage_records_source;
DROP INDEX IF EXISTS idx_usage_records_tenant_type_period;
DROP INDEX IF EXISTS idx_usage_records_period;
DROP INDEX IF EXISTS idx_usage_records_recorded_at;
DROP INDEX IF EXISTS idx_usage_records_usage_type;
DROP INDEX IF EXISTS idx_usage_records_tenant_id;

-- Drop tables
DROP TABLE IF EXISTS usage_quotas;
DROP TABLE IF EXISTS usage_records;
