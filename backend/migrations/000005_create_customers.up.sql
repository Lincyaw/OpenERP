-- Migration: Create customers table
-- Description: Creates the customers table for the partner module

CREATE TABLE customers (
    -- Primary key
    id UUID PRIMARY KEY,

    -- Multi-tenancy
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic information
    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    short_name VARCHAR(100),
    type VARCHAR(20) NOT NULL DEFAULT 'individual',
    level VARCHAR(20) NOT NULL DEFAULT 'normal',
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

    -- Business information
    tax_id VARCHAR(50),
    credit_limit DECIMAL(18,4) NOT NULL DEFAULT 0,
    balance DECIMAL(18,4) NOT NULL DEFAULT 0,

    -- Additional fields
    notes TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    attributes JSONB DEFAULT '{}',

    -- Timestamps and versioning
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT uq_customer_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT chk_customer_type CHECK (type IN ('individual', 'organization')),
    CONSTRAINT chk_customer_level CHECK (level IN ('normal', 'silver', 'gold', 'platinum', 'vip')),
    CONSTRAINT chk_customer_status CHECK (status IN ('active', 'inactive', 'suspended')),
    CONSTRAINT chk_customer_credit_limit CHECK (credit_limit >= 0),
    CONSTRAINT chk_customer_balance CHECK (balance >= 0)
);

-- Indexes for tenant isolation
CREATE INDEX idx_customer_tenant_id ON customers(tenant_id);
CREATE INDEX idx_customer_tenant_code ON customers(tenant_id, code);

-- Indexes for common queries
CREATE INDEX idx_customer_tenant_status ON customers(tenant_id, status);
CREATE INDEX idx_customer_tenant_type ON customers(tenant_id, type);
CREATE INDEX idx_customer_tenant_level ON customers(tenant_id, level);
CREATE INDEX idx_customer_tenant_phone ON customers(tenant_id, phone) WHERE phone IS NOT NULL AND phone != '';
CREATE INDEX idx_customer_tenant_email ON customers(tenant_id, email) WHERE email IS NOT NULL AND email != '';

-- Index for sorting
CREATE INDEX idx_customer_tenant_sort ON customers(tenant_id, sort_order, name);

-- Index for customers with balance
CREATE INDEX idx_customer_tenant_balance ON customers(tenant_id, balance) WHERE balance > 0;

-- Full-text search indexes for name and code
CREATE INDEX idx_customer_name_pattern ON customers(tenant_id, name varchar_pattern_ops);
CREATE INDEX idx_customer_code_pattern ON customers(tenant_id, code varchar_pattern_ops);

-- GIN index for JSONB attributes
CREATE INDEX idx_customer_attributes ON customers USING GIN (attributes);

-- Auto-update updated_at trigger
CREATE TRIGGER update_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE customers IS 'Customer master data for the partner module';
COMMENT ON COLUMN customers.code IS 'Unique customer code within tenant';
COMMENT ON COLUMN customers.type IS 'Customer type: individual or organization';
COMMENT ON COLUMN customers.level IS 'Customer tier: normal, silver, gold, platinum, vip';
COMMENT ON COLUMN customers.status IS 'Customer status: active, inactive, suspended';
COMMENT ON COLUMN customers.credit_limit IS 'Maximum credit allowed for this customer';
COMMENT ON COLUMN customers.balance IS 'Prepaid balance (deposit/recharge)';
COMMENT ON COLUMN customers.tax_id IS 'Tax identification number for invoicing';
COMMENT ON COLUMN customers.attributes IS 'Custom attributes stored as JSON';
