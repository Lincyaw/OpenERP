-- Migration: create_categories (rollback)
-- Created: 2026-01-24
-- Description: Drop categories table

-- Drop trigger
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;

-- Drop indexes
DROP INDEX IF EXISTS idx_categories_path_pattern;
DROP INDEX IF EXISTS idx_categories_tenant_status;
DROP INDEX IF EXISTS idx_categories_tenant_parent;
DROP INDEX IF EXISTS idx_categories_sort_order;
DROP INDEX IF EXISTS idx_categories_level;
DROP INDEX IF EXISTS idx_categories_status;
DROP INDEX IF EXISTS idx_categories_path;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP INDEX IF EXISTS idx_categories_tenant_id;

-- Drop table
DROP TABLE IF EXISTS categories;
