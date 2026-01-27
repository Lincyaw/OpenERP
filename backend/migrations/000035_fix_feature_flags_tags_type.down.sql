-- Rollback: Revert feature_flags tags column from JSONB to VARCHAR(100)[]

-- Drop the GIN index on JSONB
DROP INDEX IF EXISTS idx_feature_flags_tags;

-- Create temporary column
ALTER TABLE feature_flags ADD COLUMN tags_old VARCHAR(100)[];

-- Migrate data back: Convert JSONB array to PostgreSQL array
UPDATE feature_flags
SET tags_old = ARRAY(
    SELECT jsonb_array_elements_text(tags)
);

-- Drop JSONB column
ALTER TABLE feature_flags DROP COLUMN tags;

-- Rename old column back
ALTER TABLE feature_flags RENAME COLUMN tags_old TO tags;

-- Create GIN index on VARCHAR[] column
CREATE INDEX idx_feature_flags_tags ON feature_flags USING GIN(tags);

-- Update comment
COMMENT ON COLUMN feature_flags.tags IS 'Searchable tags for organizing flags';
