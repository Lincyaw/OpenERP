-- Migration: Rollback flag_overrides schema fix
-- Description: Remove updated_at column from flag_overrides table

-- Drop trigger first
DROP TRIGGER IF EXISTS trg_flag_overrides_updated_at ON flag_overrides;

-- Remove updated_at column
ALTER TABLE flag_overrides DROP COLUMN IF EXISTS updated_at;
