-- Migration: create_product_units
-- Created: 2026-01-24
-- Description: Create product_units table for multi-unit support

-- Create product_units table
CREATE TABLE IF NOT EXISTS product_units (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    unit_code VARCHAR(20) NOT NULL,
    unit_name VARCHAR(50) NOT NULL,
    conversion_rate DECIMAL(18, 6) NOT NULL,
    default_purchase_price DECIMAL(18, 4) NOT NULL DEFAULT 0,
    default_selling_price DECIMAL(18, 4) NOT NULL DEFAULT 0,
    is_default_purchase_unit BOOLEAN NOT NULL DEFAULT false,
    is_default_sales_unit BOOLEAN NOT NULL DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_product_unit_conversion_rate CHECK (conversion_rate > 0),
    CONSTRAINT chk_product_unit_purchase_price CHECK (default_purchase_price >= 0),
    CONSTRAINT chk_product_unit_selling_price CHECK (default_selling_price >= 0),
    CONSTRAINT uq_product_unit_code UNIQUE (tenant_id, product_id, unit_code)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_product_units_tenant_id ON product_units(tenant_id);
CREATE INDEX IF NOT EXISTS idx_product_units_product_id ON product_units(product_id);
CREATE INDEX IF NOT EXISTS idx_product_units_tenant_product ON product_units(tenant_id, product_id);

-- Create index for finding default units
CREATE INDEX IF NOT EXISTS idx_product_units_default_purchase ON product_units(tenant_id, product_id, is_default_purchase_unit) WHERE is_default_purchase_unit = true;
CREATE INDEX IF NOT EXISTS idx_product_units_default_sales ON product_units(tenant_id, product_id, is_default_sales_unit) WHERE is_default_sales_unit = true;

-- Create trigger for updated_at timestamp
CREATE TRIGGER update_product_units_updated_at
    BEFORE UPDATE ON product_units
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE product_units IS 'Alternate units for products with conversion rates to base unit';
COMMENT ON COLUMN product_units.product_id IS 'Reference to the product';
COMMENT ON COLUMN product_units.unit_code IS 'Unit code (e.g., BOX, CASE, PACK)';
COMMENT ON COLUMN product_units.unit_name IS 'Display name for the unit (e.g., 箱, 盒, 包)';
COMMENT ON COLUMN product_units.conversion_rate IS 'Conversion rate to base unit (1 this unit = X base units)';
COMMENT ON COLUMN product_units.default_purchase_price IS 'Default purchase price for this unit';
COMMENT ON COLUMN product_units.default_selling_price IS 'Default selling price for this unit';
COMMENT ON COLUMN product_units.is_default_purchase_unit IS 'Whether this is the default unit for purchasing';
COMMENT ON COLUMN product_units.is_default_sales_unit IS 'Whether this is the default unit for sales';
COMMENT ON COLUMN product_units.sort_order IS 'Display order in unit selection lists';
