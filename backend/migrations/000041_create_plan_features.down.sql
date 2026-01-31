-- Migration: Drop plan_features table
-- Description: Rollback plan-feature mapping implementation

-- Drop plan_features table (will cascade to triggers)
DROP TABLE IF EXISTS plan_features;
