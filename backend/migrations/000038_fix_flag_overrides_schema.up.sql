-- Migration: Fix flag_overrides schema mismatch
-- Description: Add missing updated_at column to flag_overrides table

-- Add updated_at column to flag_overrides table
ALTER TABLE flag_overrides ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Create trigger to auto-update updated_at
CREATE TRIGGER trg_flag_overrides_updated_at
    BEFORE UPDATE ON flag_overrides
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comment for updated_at column
COMMENT ON COLUMN flag_overrides.updated_at IS 'Last update timestamp';
