-- Migration: Fix feature_flags tags column type
-- Description: Change tags column from VARCHAR(100)[] to JSONB for GORM compatibility
-- Issue: GORM sends JSON array format ["tag1","tag2"] but PostgreSQL VARCHAR[] expects {tag1,tag2}

-- Drop the old GIN index on VARCHAR[]
DROP INDEX IF EXISTS idx_feature_flags_tags;

-- Create a temporary column to preserve data
ALTER TABLE feature_flags ADD COLUMN tags_new JSONB DEFAULT '[]';

-- Migrate existing data: Convert PostgreSQL array to JSONB array
UPDATE feature_flags
SET tags_new = CASE
    WHEN tags IS NULL OR array_length(tags, 1) IS NULL THEN '[]'::jsonb
    ELSE array_to_json(tags)::jsonb
END;

-- Drop the old VARCHAR[] column
ALTER TABLE feature_flags DROP COLUMN tags;

-- Rename the new column
ALTER TABLE feature_flags RENAME COLUMN tags_new TO tags;

-- Set NOT NULL constraint with default
ALTER TABLE feature_flags ALTER COLUMN tags SET DEFAULT '[]';
ALTER TABLE feature_flags ALTER COLUMN tags SET NOT NULL;

-- Create new GIN index on JSONB column for efficient querying
CREATE INDEX idx_feature_flags_tags ON feature_flags USING GIN(tags);

-- Update comment
COMMENT ON COLUMN feature_flags.tags IS 'Searchable tags for organizing flags (JSONB array)';
