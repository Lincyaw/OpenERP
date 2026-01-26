-- Migration: Create customer_levels table
-- Description: Creates the customer_levels table to support customizable customer tiers per tenant

CREATE TABLE customer_levels (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Multi-tenancy
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Level definition
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    discount_rate DECIMAL(5,4) NOT NULL DEFAULT 0,  -- Stored as decimal (e.g., 0.05 for 5%)

    -- Display and ordering
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,  -- One default level per tenant
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Metadata
    description TEXT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT uq_customer_level_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT chk_customer_level_discount_rate CHECK (discount_rate >= 0 AND discount_rate <= 1)
);

-- Indexes
CREATE INDEX idx_customer_level_tenant_id ON customer_levels(tenant_id);
CREATE INDEX idx_customer_level_tenant_active ON customer_levels(tenant_id, is_active);
CREATE INDEX idx_customer_level_tenant_sort ON customer_levels(tenant_id, sort_order);
CREATE INDEX idx_customer_level_tenant_default ON customer_levels(tenant_id, is_default) WHERE is_default = TRUE;

-- Auto-update updated_at trigger
CREATE TRIGGER update_customer_levels_updated_at
    BEFORE UPDATE ON customer_levels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to ensure only one default level per tenant
CREATE OR REPLACE FUNCTION ensure_single_default_customer_level()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_default = TRUE THEN
        UPDATE customer_levels
        SET is_default = FALSE
        WHERE tenant_id = NEW.tenant_id
          AND id != NEW.id
          AND is_default = TRUE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ensure_single_default_customer_level_trigger
    BEFORE INSERT OR UPDATE ON customer_levels
    FOR EACH ROW
    EXECUTE FUNCTION ensure_single_default_customer_level();

-- Alter customers table to reference customer_levels
-- First, add the level_id column
ALTER TABLE customers ADD COLUMN level_id UUID REFERENCES customer_levels(id);

-- Create index for the foreign key
CREATE INDEX idx_customer_level_id ON customers(level_id);

-- Insert default customer levels for the default tenant (00000000-0000-0000-0000-000000000001)
INSERT INTO customer_levels (id, tenant_id, code, name, discount_rate, sort_order, is_default, is_active, description) VALUES
('60000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'normal', '普通会员', 0.0000, 0, true, true, '普通会员，无折扣'),
('60000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'silver', '银卡会员', 0.0300, 1, false, true, '银卡会员，享受3%折扣'),
('60000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'gold', '金卡会员', 0.0500, 2, false, true, '金卡会员，享受5%折扣'),
('60000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'platinum', '白金会员', 0.0800, 3, false, true, '白金会员，享受8%折扣'),
('60000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'vip', 'VIP会员', 0.1000, 4, false, true, 'VIP会员，享受10%折扣')
ON CONFLICT DO NOTHING;

-- Update existing customers to set level_id based on their current level code
UPDATE customers c
SET level_id = cl.id
FROM customer_levels cl
WHERE c.tenant_id = cl.tenant_id
  AND c.level = cl.code
  AND c.level_id IS NULL;

-- Comments
COMMENT ON TABLE customer_levels IS 'Customer level/tier definitions with tenant-specific discount rates';
COMMENT ON COLUMN customer_levels.code IS 'Unique level code within tenant (e.g., normal, silver, gold)';
COMMENT ON COLUMN customer_levels.name IS 'Display name for the customer level';
COMMENT ON COLUMN customer_levels.discount_rate IS 'Discount rate as decimal (0.05 = 5% discount)';
COMMENT ON COLUMN customer_levels.sort_order IS 'Display order (lower = higher tier)';
COMMENT ON COLUMN customer_levels.is_default IS 'Whether this is the default level for new customers';
COMMENT ON COLUMN customer_levels.is_active IS 'Whether this level is currently available';
COMMENT ON COLUMN customers.level_id IS 'Reference to customer_levels table (nullable during migration)';
