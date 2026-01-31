-- Migration: Drop plan_feature_change_logs table
-- Description: Rollback audit log for plan feature configuration changes

DROP TABLE IF EXISTS plan_feature_change_logs;
