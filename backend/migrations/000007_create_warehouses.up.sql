-- Migration: 000007_create_warehouses
-- Description: Create warehouses table for partner context

BEGIN;

-- Create warehouses table
CREATE TABLE IF NOT EXISTS warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic information
    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    short_name VARCHAR(100),
    type VARCHAR(20) NOT NULL DEFAULT 'physical',
    status VARCHAR(20) NOT NULL DEFAULT 'active',

    -- Contact information
    contact_name VARCHAR(100),
    phone VARCHAR(50),
    email VARCHAR(200),

    -- Address information
    address TEXT,
    city VARCHAR(100),
    province VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100) DEFAULT '中国',

    -- Warehouse-specific fields
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    capacity INTEGER NOT NULL DEFAULT 0,

    -- Additional fields
    notes TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    attributes JSONB DEFAULT '{}',

    -- Audit fields
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT warehouses_type_check CHECK (type IN ('physical', 'virtual', 'consign', 'transit')),
    CONSTRAINT warehouses_status_check CHECK (status IN ('active', 'inactive')),
    CONSTRAINT warehouses_capacity_check CHECK (capacity >= 0)
);

-- Create unique constraint on tenant_id + code
CREATE UNIQUE INDEX idx_warehouse_tenant_code ON warehouses(tenant_id, code);

-- Create partial unique index to ensure only one default warehouse per tenant
CREATE UNIQUE INDEX idx_warehouses_tenant_default ON warehouses(tenant_id) WHERE is_default = TRUE;

-- Create indexes for common queries
CREATE INDEX idx_warehouses_tenant ON warehouses(tenant_id);
CREATE INDEX idx_warehouses_status ON warehouses(tenant_id, status);
CREATE INDEX idx_warehouses_type ON warehouses(tenant_id, type);
CREATE INDEX idx_warehouses_is_default ON warehouses(tenant_id, is_default) WHERE is_default = TRUE;
CREATE INDEX idx_warehouses_sort_order ON warehouses(tenant_id, sort_order);
CREATE INDEX idx_warehouses_created_at ON warehouses(tenant_id, created_at DESC);

-- Create GIN index for JSONB attributes
CREATE INDEX idx_warehouses_attributes ON warehouses USING GIN (attributes);

-- Create pattern indexes for ILIKE searches
CREATE INDEX idx_warehouses_name_pattern ON warehouses(tenant_id, name varchar_pattern_ops);
CREATE INDEX idx_warehouses_code_pattern ON warehouses(tenant_id, code varchar_pattern_ops);

-- Create updated_at trigger
CREATE TRIGGER update_warehouses_updated_at
    BEFORE UPDATE ON warehouses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE warehouses IS 'Stores warehouse/location information for the partner context';
COMMENT ON COLUMN warehouses.code IS 'Unique warehouse code within a tenant (e.g., WH001)';
COMMENT ON COLUMN warehouses.name IS 'Full warehouse name';
COMMENT ON COLUMN warehouses.short_name IS 'Abbreviated name for display';
COMMENT ON COLUMN warehouses.type IS 'Warehouse type: physical, virtual, consign, transit';
COMMENT ON COLUMN warehouses.status IS 'Warehouse status: active, inactive';
COMMENT ON COLUMN warehouses.contact_name IS 'Warehouse manager or contact person';
COMMENT ON COLUMN warehouses.is_default IS 'Whether this is the default warehouse for operations';
COMMENT ON COLUMN warehouses.capacity IS 'Storage capacity in units (0 = unlimited)';
COMMENT ON COLUMN warehouses.attributes IS 'Custom attributes stored as JSON';

COMMIT;
