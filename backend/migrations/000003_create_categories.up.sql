-- Migration: create_categories
-- Created: 2026-01-24
-- Description: Create categories table for product classification with tree structure support

-- Create categories table
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    path VARCHAR(500) NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT chk_category_status CHECK (status IN ('active', 'inactive')),
    CONSTRAINT chk_category_level CHECK (level >= 0 AND level < 5),
    CONSTRAINT uq_category_tenant_code UNIQUE (tenant_id, code)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_categories_tenant_id ON categories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_path ON categories(path);
CREATE INDEX IF NOT EXISTS idx_categories_status ON categories(status);
CREATE INDEX IF NOT EXISTS idx_categories_level ON categories(level);
CREATE INDEX IF NOT EXISTS idx_categories_sort_order ON categories(sort_order);

-- Create composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_categories_tenant_parent ON categories(tenant_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_tenant_status ON categories(tenant_id, status);

-- Create index for path prefix matching (tree traversal)
CREATE INDEX IF NOT EXISTS idx_categories_path_pattern ON categories(tenant_id, path varchar_pattern_ops);

-- Create trigger for updated_at timestamp
CREATE TRIGGER update_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE categories IS 'Product categories with tree structure support';
COMMENT ON COLUMN categories.code IS 'Unique category code within tenant (uppercase)';
COMMENT ON COLUMN categories.name IS 'Display name of the category';
COMMENT ON COLUMN categories.parent_id IS 'Reference to parent category (NULL for root categories)';
COMMENT ON COLUMN categories.path IS 'Materialized path for efficient tree queries (format: id1/id2/id3)';
COMMENT ON COLUMN categories.level IS 'Depth level in the tree (0 for root, max 4)';
COMMENT ON COLUMN categories.sort_order IS 'Display order among siblings';
COMMENT ON COLUMN categories.status IS 'Category status (active/inactive)';
