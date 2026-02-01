-- Migration: create_auto_print_rules (down)
-- Created: 2026-02-01
-- Description: Drop auto_print_rules table

-- Drop trigger
DROP TRIGGER IF EXISTS update_auto_print_rules_updated_at ON auto_print_rules;

-- Drop indexes
DROP INDEX IF EXISTS idx_auto_print_rules_enabled;
DROP INDEX IF EXISTS idx_auto_print_rules_lookup;
DROP INDEX IF EXISTS idx_auto_print_rules_tenant;

-- Drop table
DROP TABLE IF EXISTS auto_print_rules;
