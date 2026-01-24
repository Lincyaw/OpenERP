-- Migration: Create stock_takings and stock_taking_items tables
-- These tables support the stock taking (inventory count) functionality

-- Create stock_takings table (aggregate root)
CREATE TABLE stock_takings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    taking_number VARCHAR(50) NOT NULL,
    warehouse_id UUID NOT NULL,
    warehouse_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'COUNTING', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'CANCELLED')),
    taking_date DATE NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    approved_by_id UUID,
    approved_by_name VARCHAR(100),
    created_by_id UUID NOT NULL,
    created_by_name VARCHAR(100) NOT NULL,
    total_items INTEGER NOT NULL DEFAULT 0 CHECK (total_items >= 0),
    counted_items INTEGER NOT NULL DEFAULT 0 CHECK (counted_items >= 0),
    difference_items INTEGER NOT NULL DEFAULT 0 CHECK (difference_items >= 0),
    total_difference DECIMAL(18, 4) NOT NULL DEFAULT 0,
    approval_note VARCHAR(500),
    remark VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique taking number per tenant
    CONSTRAINT idx_stock_taking_number_tenant UNIQUE (tenant_id, taking_number),

    -- Foreign keys
    CONSTRAINT fk_stock_taking_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT fk_stock_taking_warehouse FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE RESTRICT,
    CONSTRAINT fk_stock_taking_approver FOREIGN KEY (approved_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_stock_taking_creator FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_stock_taking_tenant ON stock_takings(tenant_id);
CREATE INDEX idx_stock_taking_warehouse ON stock_takings(warehouse_id);
CREATE INDEX idx_stock_taking_status ON stock_takings(tenant_id, status);
CREATE INDEX idx_stock_taking_date ON stock_takings(tenant_id, taking_date);
CREATE INDEX idx_stock_taking_pending ON stock_takings(tenant_id) WHERE status = 'PENDING_APPROVAL';

-- Add update trigger for updated_at
CREATE TRIGGER trg_stock_takings_updated_at
    BEFORE UPDATE ON stock_takings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create stock_taking_items table
CREATE TABLE stock_taking_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stock_taking_id UUID NOT NULL,
    product_id UUID NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    system_quantity DECIMAL(18, 4) NOT NULL CHECK (system_quantity >= 0),
    actual_quantity DECIMAL(18, 4),
    difference_qty DECIMAL(18, 4),
    unit_cost DECIMAL(18, 4) NOT NULL CHECK (unit_cost >= 0),
    difference_amount DECIMAL(18, 4),
    counted BOOLEAN NOT NULL DEFAULT FALSE,
    remark VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique product per stock taking
    CONSTRAINT idx_stock_taking_item_product UNIQUE (stock_taking_id, product_id),

    -- Foreign keys
    CONSTRAINT fk_stock_taking_item_taking FOREIGN KEY (stock_taking_id) REFERENCES stock_takings(id) ON DELETE CASCADE,
    CONSTRAINT fk_stock_taking_item_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
);

-- Create indexes for stock_taking_items
CREATE INDEX idx_stock_taking_item_taking ON stock_taking_items(stock_taking_id);
CREATE INDEX idx_stock_taking_item_product_id ON stock_taking_items(product_id);
CREATE INDEX idx_stock_taking_item_uncounted ON stock_taking_items(stock_taking_id) WHERE counted = FALSE;
CREATE INDEX idx_stock_taking_item_difference ON stock_taking_items(stock_taking_id) WHERE counted = TRUE AND difference_qty <> 0;

-- Add update trigger for stock_taking_items
CREATE TRIGGER trg_stock_taking_items_updated_at
    BEFORE UPDATE ON stock_taking_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE stock_takings IS 'Stock taking (inventory count) documents';
COMMENT ON COLUMN stock_takings.taking_number IS 'Unique document number for the stock taking';
COMMENT ON COLUMN stock_takings.status IS 'Document status: DRAFT, COUNTING, PENDING_APPROVAL, APPROVED, REJECTED, CANCELLED';
COMMENT ON COLUMN stock_takings.taking_date IS 'Date when the stock taking was performed';
COMMENT ON COLUMN stock_takings.started_at IS 'Timestamp when counting started';
COMMENT ON COLUMN stock_takings.completed_at IS 'Timestamp when counting was completed';
COMMENT ON COLUMN stock_takings.approved_at IS 'Timestamp when approved or rejected';
COMMENT ON COLUMN stock_takings.total_items IS 'Total number of items to count';
COMMENT ON COLUMN stock_takings.counted_items IS 'Number of items that have been counted';
COMMENT ON COLUMN stock_takings.difference_items IS 'Number of items with quantity difference';
COMMENT ON COLUMN stock_takings.total_difference IS 'Total monetary value of differences';

COMMENT ON TABLE stock_taking_items IS 'Individual items in a stock taking document';
COMMENT ON COLUMN stock_taking_items.system_quantity IS 'Quantity recorded in the system at time of count';
COMMENT ON COLUMN stock_taking_items.actual_quantity IS 'Actual physical count quantity';
COMMENT ON COLUMN stock_taking_items.difference_qty IS 'Difference between actual and system (actual - system)';
COMMENT ON COLUMN stock_taking_items.unit_cost IS 'Unit cost at time of count for valuation';
COMMENT ON COLUMN stock_taking_items.difference_amount IS 'Monetary value of difference (difference_qty * unit_cost)';
COMMENT ON COLUMN stock_taking_items.counted IS 'Whether this item has been physically counted';
