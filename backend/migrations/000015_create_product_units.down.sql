-- Migration: create_product_units (down)
-- Created: 2026-01-24
-- Description: Drop product_units table

DROP TRIGGER IF EXISTS update_product_units_updated_at ON product_units;

DROP INDEX IF EXISTS idx_product_units_tenant_id;
DROP INDEX IF EXISTS idx_product_units_product_id;
DROP INDEX IF EXISTS idx_product_units_tenant_product;
DROP INDEX IF EXISTS idx_product_units_default_purchase;
DROP INDEX IF EXISTS idx_product_units_default_sales;

DROP TABLE IF EXISTS product_units;
