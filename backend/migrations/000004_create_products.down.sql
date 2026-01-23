-- Migration: create_products (rollback)
-- Description: Drop products table and related objects

-- Drop trigger
DROP TRIGGER IF EXISTS update_products_updated_at ON products;

-- Drop indexes
DROP INDEX IF EXISTS idx_products_attributes;
DROP INDEX IF EXISTS idx_products_code_pattern;
DROP INDEX IF EXISTS idx_products_name_pattern;
DROP INDEX IF EXISTS idx_products_tenant_barcode;
DROP INDEX IF EXISTS idx_products_tenant_category;
DROP INDEX IF EXISTS idx_products_tenant_status;
DROP INDEX IF EXISTS idx_products_sort_order;
DROP INDEX IF EXISTS idx_products_status;
DROP INDEX IF EXISTS idx_products_barcode;
DROP INDEX IF EXISTS idx_products_category_id;
DROP INDEX IF EXISTS idx_products_tenant_id;

-- Drop table
DROP TABLE IF EXISTS products;
