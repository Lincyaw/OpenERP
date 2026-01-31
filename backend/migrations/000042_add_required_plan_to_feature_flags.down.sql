-- Rollback: Remove required_plan column from feature_flags table

-- Drop the index first
DROP INDEX IF EXISTS idx_feature_flags_required_plan;

-- Remove the column
ALTER TABLE feature_flags DROP COLUMN IF EXISTS required_plan;
