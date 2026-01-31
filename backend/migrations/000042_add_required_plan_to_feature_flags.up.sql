-- Migration: Add required_plan column to feature_flags table
-- Description: Enables feature flag integration with subscription plans

-- Add required_plan column to feature_flags table
ALTER TABLE feature_flags
ADD COLUMN required_plan VARCHAR(20) DEFAULT '' CHECK (required_plan IN ('', 'free', 'basic', 'pro', 'enterprise'));

-- Create index for filtering by required_plan
CREATE INDEX idx_feature_flags_required_plan ON feature_flags(required_plan) WHERE required_plan != '';

-- Add comment for the new column
COMMENT ON COLUMN feature_flags.required_plan IS 'Minimum subscription plan required to access this feature (empty = no restriction)';
