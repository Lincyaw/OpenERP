-- Migration: create_products
-- Created: 2026-01-24
-- Description: Create products table for inventory management

-- Create products table
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    barcode VARCHAR(50),
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    unit VARCHAR(20) NOT NULL,
    purchase_price DECIMAL(18, 4) NOT NULL DEFAULT 0,
    selling_price DECIMAL(18, 4) NOT NULL DEFAULT 0,
    min_stock DECIMAL(18, 4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    sort_order INTEGER NOT NULL DEFAULT 0,
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT chk_product_status CHECK (status IN ('active', 'inactive', 'discontinued')),
    CONSTRAINT chk_product_purchase_price CHECK (purchase_price >= 0),
    CONSTRAINT chk_product_selling_price CHECK (selling_price >= 0),
    CONSTRAINT chk_product_min_stock CHECK (min_stock >= 0),
    CONSTRAINT uq_product_tenant_code UNIQUE (tenant_id, code)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_products_tenant_id ON products(tenant_id);
CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);
CREATE INDEX IF NOT EXISTS idx_products_sort_order ON products(sort_order);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_products_tenant_status ON products(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_products_tenant_category ON products(tenant_id, category_id);
CREATE INDEX IF NOT EXISTS idx_products_tenant_barcode ON products(tenant_id, barcode) WHERE barcode IS NOT NULL AND barcode != '';

-- Create index for text search on name and code
CREATE INDEX IF NOT EXISTS idx_products_name_pattern ON products(name varchar_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_products_code_pattern ON products(code varchar_pattern_ops);

-- Create GIN index for JSONB attributes
CREATE INDEX IF NOT EXISTS idx_products_attributes ON products USING GIN (attributes);

-- Create trigger for updated_at timestamp
CREATE TRIGGER update_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE products IS 'Products/SKU master data for inventory management';
COMMENT ON COLUMN products.code IS 'Unique product code/SKU within tenant (uppercase)';
COMMENT ON COLUMN products.name IS 'Display name of the product';
COMMENT ON COLUMN products.description IS 'Detailed product description';
COMMENT ON COLUMN products.barcode IS 'Product barcode (optional)';
COMMENT ON COLUMN products.category_id IS 'Reference to product category';
COMMENT ON COLUMN products.unit IS 'Base unit of measurement (e.g., pcs, kg, box)';
COMMENT ON COLUMN products.purchase_price IS 'Default purchase/cost price (4 decimal places)';
COMMENT ON COLUMN products.selling_price IS 'Default selling price (4 decimal places)';
COMMENT ON COLUMN products.min_stock IS 'Minimum stock level for alerts (4 decimal places)';
COMMENT ON COLUMN products.status IS 'Product status (active/inactive/discontinued)';
COMMENT ON COLUMN products.sort_order IS 'Display order in listings';
COMMENT ON COLUMN products.attributes IS 'Custom product attributes as JSON';
