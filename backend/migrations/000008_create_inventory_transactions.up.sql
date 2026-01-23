-- Migration: Create inventory_transactions table
-- This table records all inventory movements and cannot be modified once created (append-only audit log)

-- Create transaction type enum
CREATE TYPE transaction_type AS ENUM (
    'INBOUND',
    'OUTBOUND',
    'ADJUSTMENT_INCREASE',
    'ADJUSTMENT_DECREASE',
    'TRANSFER_IN',
    'TRANSFER_OUT',
    'RETURN',
    'LOCK',
    'UNLOCK'
);

-- Create source type enum
CREATE TYPE source_type AS ENUM (
    'PURCHASE_ORDER',
    'SALES_ORDER',
    'SALES_RETURN',
    'PURCHASE_RETURN',
    'STOCK_TAKING',
    'MANUAL_ADJUSTMENT',
    'TRANSFER',
    'INITIAL_STOCK'
);

-- Create inventory_transactions table
CREATE TABLE inventory_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    inventory_item_id UUID NOT NULL,
    warehouse_id UUID NOT NULL,
    product_id UUID NOT NULL,
    transaction_type transaction_type NOT NULL,
    quantity DECIMAL(18, 4) NOT NULL CHECK (quantity > 0),
    unit_cost DECIMAL(18, 4) NOT NULL CHECK (unit_cost >= 0),
    total_cost DECIMAL(18, 4) NOT NULL CHECK (total_cost >= 0),
    balance_before DECIMAL(18, 4) NOT NULL CHECK (balance_before >= 0),
    balance_after DECIMAL(18, 4) NOT NULL CHECK (balance_after >= 0),
    source_type source_type NOT NULL,
    source_id VARCHAR(50) NOT NULL,
    source_line_id VARCHAR(50),
    batch_id UUID,
    lock_id UUID,
    reference VARCHAR(100),
    reason VARCHAR(255),
    operator_id UUID,
    transaction_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Tenant isolation
    CONSTRAINT fk_inv_tx_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_inv_tx_tenant_time ON inventory_transactions(tenant_id, transaction_date DESC);
CREATE INDEX idx_inv_tx_item ON inventory_transactions(inventory_item_id);
CREATE INDEX idx_inv_tx_warehouse ON inventory_transactions(warehouse_id);
CREATE INDEX idx_inv_tx_product ON inventory_transactions(product_id);
CREATE INDEX idx_inv_tx_type ON inventory_transactions(transaction_type);
CREATE INDEX idx_inv_tx_source ON inventory_transactions(source_type, source_id);
CREATE INDEX idx_inv_tx_batch ON inventory_transactions(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_inv_tx_lock ON inventory_transactions(lock_id) WHERE lock_id IS NOT NULL;
CREATE INDEX idx_inv_tx_operator ON inventory_transactions(operator_id) WHERE operator_id IS NOT NULL;

-- Add trigger to prevent updates (append-only)
CREATE OR REPLACE FUNCTION prevent_inventory_transaction_update()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Inventory transactions cannot be modified. Create a new transaction for corrections.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_inv_tx_update
    BEFORE UPDATE ON inventory_transactions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_inventory_transaction_update();

-- Add trigger to prevent deletes (immutable audit log)
CREATE OR REPLACE FUNCTION prevent_inventory_transaction_delete()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Inventory transactions cannot be deleted. This is an immutable audit log.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_prevent_inv_tx_delete
    BEFORE DELETE ON inventory_transactions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_inventory_transaction_delete();

-- Add comments for documentation
COMMENT ON TABLE inventory_transactions IS 'Immutable audit log of all inventory movements';
COMMENT ON COLUMN inventory_transactions.quantity IS 'Always positive - direction determined by transaction_type';
COMMENT ON COLUMN inventory_transactions.balance_before IS 'Available quantity before this transaction';
COMMENT ON COLUMN inventory_transactions.balance_after IS 'Available quantity after this transaction';
COMMENT ON COLUMN inventory_transactions.source_type IS 'Type of document that triggered this transaction';
COMMENT ON COLUMN inventory_transactions.source_id IS 'ID of the source document';
COMMENT ON COLUMN inventory_transactions.source_line_id IS 'ID of the source line item (for multi-line documents)';
