-- Migration: create_product_attachments
-- Created: 2026-01-29
-- Description: Create product_attachments table for file storage metadata

-- Create product_attachments table
CREATE TABLE IF NOT EXISTS product_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    thumbnail_key VARCHAR(500),
    sort_order INTEGER NOT NULL DEFAULT 0,
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Check constraints for type enum (matches domain AttachmentType)
    CONSTRAINT chk_product_attachment_type CHECK (type IN (
        'main_image',
        'gallery_image',
        'document',
        'other'
    )),

    -- Check constraints for status enum (matches domain AttachmentStatus)
    CONSTRAINT chk_product_attachment_status CHECK (status IN (
        'pending',
        'active',
        'deleted'
    )),

    -- File size must be positive and within limit (100MB)
    CONSTRAINT chk_product_attachment_file_size CHECK (file_size > 0 AND file_size <= 104857600),

    -- Sort order must be non-negative
    CONSTRAINT chk_product_attachment_sort_order CHECK (sort_order >= 0)
);

-- Primary indexes for tenant isolation and product lookup
CREATE INDEX IF NOT EXISTS idx_product_attachments_tenant_id ON product_attachments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_product_attachments_product_id ON product_attachments(product_id);

-- Composite index for the most common query: find attachments by tenant and product
CREATE INDEX IF NOT EXISTS idx_product_attachments_tenant_product ON product_attachments(tenant_id, product_id);

-- Index for filtering by status (typically query active attachments)
CREATE INDEX IF NOT EXISTS idx_product_attachments_status ON product_attachments(status);

-- Composite index for finding active attachments by product (common query pattern)
CREATE INDEX IF NOT EXISTS idx_product_attachments_product_active ON product_attachments(tenant_id, product_id, status)
    WHERE status = 'active';

-- Partial index to optimize main image lookup (only one main image per product)
-- This is a critical query for product listing pages
CREATE INDEX IF NOT EXISTS idx_product_attachments_main_image ON product_attachments(tenant_id, product_id)
    WHERE type = 'main_image' AND status = 'active';

-- Index for finding attachments by type within a product
CREATE INDEX IF NOT EXISTS idx_product_attachments_type ON product_attachments(tenant_id, product_id, type)
    WHERE status = 'active';

-- Index for uploaded_by to track user uploads
CREATE INDEX IF NOT EXISTS idx_product_attachments_uploaded_by ON product_attachments(uploaded_by)
    WHERE uploaded_by IS NOT NULL;

-- Index for sort order to enable efficient ordering
CREATE INDEX IF NOT EXISTS idx_product_attachments_sort ON product_attachments(tenant_id, product_id, sort_order)
    WHERE status = 'active';

-- Trigger for updated_at timestamp
CREATE TRIGGER update_product_attachments_updated_at
    BEFORE UPDATE ON product_attachments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Table and column comments for documentation
COMMENT ON TABLE product_attachments IS 'File attachments associated with products (images, documents, etc.)';
COMMENT ON COLUMN product_attachments.tenant_id IS 'Reference to tenant for multi-tenancy isolation';
COMMENT ON COLUMN product_attachments.product_id IS 'Reference to the parent product';
COMMENT ON COLUMN product_attachments.type IS 'Attachment type: main_image, gallery_image, document, other';
COMMENT ON COLUMN product_attachments.status IS 'Attachment status: pending (uploading), active (confirmed), deleted (soft-deleted)';
COMMENT ON COLUMN product_attachments.file_name IS 'Original file name (sanitized, max 255 chars)';
COMMENT ON COLUMN product_attachments.file_size IS 'File size in bytes (max 100MB)';
COMMENT ON COLUMN product_attachments.content_type IS 'MIME type (e.g., image/jpeg, application/pdf)';
COMMENT ON COLUMN product_attachments.storage_key IS 'Object storage key/path (relative path in bucket)';
COMMENT ON COLUMN product_attachments.thumbnail_key IS 'Object storage key for thumbnail (images only)';
COMMENT ON COLUMN product_attachments.sort_order IS 'Display order within the product (0-based)';
COMMENT ON COLUMN product_attachments.uploaded_by IS 'User who uploaded the file (nullable for system uploads)';
