-- Migration: create_product_attachments (down)
-- Created: 2026-01-29
-- Description: Drop product_attachments table and related objects

-- Drop trigger first
DROP TRIGGER IF EXISTS update_product_attachments_updated_at ON product_attachments;

-- Drop all indexes (will be dropped with table, but explicit for clarity)
DROP INDEX IF EXISTS idx_product_attachments_sort;
DROP INDEX IF EXISTS idx_product_attachments_uploaded_by;
DROP INDEX IF EXISTS idx_product_attachments_type;
DROP INDEX IF EXISTS idx_product_attachments_main_image;
DROP INDEX IF EXISTS idx_product_attachments_product_active;
DROP INDEX IF EXISTS idx_product_attachments_status;
DROP INDEX IF EXISTS idx_product_attachments_tenant_product;
DROP INDEX IF EXISTS idx_product_attachments_product_id;
DROP INDEX IF EXISTS idx_product_attachments_tenant_id;

-- Drop the table
DROP TABLE IF EXISTS product_attachments;
