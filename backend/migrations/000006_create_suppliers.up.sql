-- Migration: 000006_create_suppliers
-- Description: Create suppliers table for partner context

BEGIN;

-- Create suppliers table
CREATE TABLE IF NOT EXISTS suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic information
    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    short_name VARCHAR(100),
    type VARCHAR(20) NOT NULL DEFAULT 'distributor',
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

    -- Financial information
    tax_id VARCHAR(50),
    bank_name VARCHAR(200),
    bank_account VARCHAR(100),
    credit_days INTEGER NOT NULL DEFAULT 0,
    credit_limit DECIMAL(18, 4) NOT NULL DEFAULT 0,
    balance DECIMAL(18, 4) NOT NULL DEFAULT 0,

    -- Additional fields
    rating INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    attributes JSONB DEFAULT '{}',

    -- Audit fields
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT suppliers_type_check CHECK (type IN ('manufacturer', 'distributor', 'retailer', 'service')),
    CONSTRAINT suppliers_status_check CHECK (status IN ('active', 'inactive', 'blocked')),
    CONSTRAINT suppliers_credit_days_check CHECK (credit_days >= 0 AND credit_days <= 365),
    CONSTRAINT suppliers_credit_limit_check CHECK (credit_limit >= 0),
    CONSTRAINT suppliers_balance_check CHECK (balance >= 0),
    CONSTRAINT suppliers_rating_check CHECK (rating >= 0 AND rating <= 5)
);

-- Create unique constraint on tenant_id + code
CREATE UNIQUE INDEX idx_supplier_tenant_code ON suppliers(tenant_id, code);

-- Create indexes for common queries
CREATE INDEX idx_suppliers_tenant ON suppliers(tenant_id);
CREATE INDEX idx_suppliers_status ON suppliers(tenant_id, status);
CREATE INDEX idx_suppliers_type ON suppliers(tenant_id, type);
CREATE INDEX idx_suppliers_phone ON suppliers(tenant_id, phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_suppliers_email ON suppliers(tenant_id, email) WHERE email IS NOT NULL;
CREATE INDEX idx_suppliers_sort_order ON suppliers(tenant_id, sort_order);
CREATE INDEX idx_suppliers_created_at ON suppliers(tenant_id, created_at DESC);

-- Create GIN index for JSONB attributes
CREATE INDEX idx_suppliers_attributes ON suppliers USING GIN (attributes);

-- Create pattern indexes for ILIKE searches
CREATE INDEX idx_suppliers_name_pattern ON suppliers(tenant_id, name varchar_pattern_ops);
CREATE INDEX idx_suppliers_code_pattern ON suppliers(tenant_id, code varchar_pattern_ops);

-- Create index for balance queries
CREATE INDEX idx_suppliers_balance ON suppliers(tenant_id, balance) WHERE balance > 0;

-- Create index for over-credit-limit queries
CREATE INDEX idx_suppliers_over_credit ON suppliers(tenant_id) WHERE balance > credit_limit AND credit_limit > 0;

-- Create updated_at trigger
CREATE TRIGGER update_suppliers_updated_at
    BEFORE UPDATE ON suppliers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE suppliers IS 'Stores supplier/vendor information for the partner context';
COMMENT ON COLUMN suppliers.code IS 'Unique supplier code within a tenant (e.g., SUP001)';
COMMENT ON COLUMN suppliers.name IS 'Full supplier name';
COMMENT ON COLUMN suppliers.short_name IS 'Abbreviated name for display';
COMMENT ON COLUMN suppliers.type IS 'Supplier type: manufacturer, distributor, retailer, service';
COMMENT ON COLUMN suppliers.status IS 'Supplier status: active, inactive, blocked';
COMMENT ON COLUMN suppliers.contact_name IS 'Primary contact person name';
COMMENT ON COLUMN suppliers.tax_id IS 'Tax identification number for invoicing';
COMMENT ON COLUMN suppliers.bank_name IS 'Bank name for payment';
COMMENT ON COLUMN suppliers.bank_account IS 'Bank account number';
COMMENT ON COLUMN suppliers.credit_days IS 'Payment terms: number of days until payment is due';
COMMENT ON COLUMN suppliers.credit_limit IS 'Maximum credit amount allowed';
COMMENT ON COLUMN suppliers.balance IS 'Current accounts payable balance';
COMMENT ON COLUMN suppliers.rating IS 'Supplier rating from 0 to 5';
COMMENT ON COLUMN suppliers.attributes IS 'Custom attributes stored as JSON';

COMMIT;
