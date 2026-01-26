-- Migration: Drop customer_levels table
-- Description: Reverts the customer_levels table creation

-- Remove the foreign key column from customers
DROP INDEX IF EXISTS idx_customer_level_id;
ALTER TABLE customers DROP COLUMN IF EXISTS level_id;

-- Drop triggers
DROP TRIGGER IF EXISTS ensure_single_default_customer_level_trigger ON customer_levels;
DROP FUNCTION IF EXISTS ensure_single_default_customer_level();

DROP TRIGGER IF EXISTS update_customer_levels_updated_at ON customer_levels;

-- Drop indexes
DROP INDEX IF EXISTS idx_customer_level_tenant_default;
DROP INDEX IF EXISTS idx_customer_level_tenant_sort;
DROP INDEX IF EXISTS idx_customer_level_tenant_active;
DROP INDEX IF EXISTS idx_customer_level_tenant_id;

-- Drop table
DROP TABLE IF EXISTS customer_levels;
