-- Migration: 000007_create_warehouses
-- Description: Drop warehouses table

BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS update_warehouses_updated_at ON warehouses;

-- Drop indexes
DROP INDEX IF EXISTS idx_warehouses_code_pattern;
DROP INDEX IF EXISTS idx_warehouses_name_pattern;
DROP INDEX IF EXISTS idx_warehouses_attributes;
DROP INDEX IF EXISTS idx_warehouses_created_at;
DROP INDEX IF EXISTS idx_warehouses_sort_order;
DROP INDEX IF EXISTS idx_warehouses_is_default;
DROP INDEX IF EXISTS idx_warehouses_type;
DROP INDEX IF EXISTS idx_warehouses_status;
DROP INDEX IF EXISTS idx_warehouses_tenant;
DROP INDEX IF EXISTS idx_warehouses_tenant_default;
DROP INDEX IF EXISTS idx_warehouse_tenant_code;

-- Drop table
DROP TABLE IF EXISTS warehouses;

COMMIT;
