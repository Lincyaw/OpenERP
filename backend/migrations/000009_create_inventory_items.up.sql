-- Migration: Create inventory_items, stock_batches, and stock_locks tables
-- These tables form the core of the inventory management system

-- Create inventory_items table (aggregate root)
CREATE TABLE inventory_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    warehouse_id UUID NOT NULL,
    product_id UUID NOT NULL,
    available_quantity DECIMAL(18, 4) NOT NULL DEFAULT 0 CHECK (available_quantity >= 0),
    locked_quantity DECIMAL(18, 4) NOT NULL DEFAULT 0 CHECK (locked_quantity >= 0),
    unit_cost DECIMAL(18, 4) NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
    min_quantity DECIMAL(18, 4) NOT NULL DEFAULT 0 CHECK (min_quantity >= 0),
    max_quantity DECIMAL(18, 4) NOT NULL DEFAULT 0 CHECK (max_quantity >= 0),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique warehouse-product combination per tenant
    CONSTRAINT idx_inventory_item_warehouse_product UNIQUE (tenant_id, warehouse_id, product_id),

    -- Foreign keys
    CONSTRAINT fk_inv_item_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT fk_inv_item_warehouse FOREIGN KEY (warehouse_id) REFERENCES warehouses(id) ON DELETE RESTRICT,
    CONSTRAINT fk_inv_item_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_inv_item_tenant ON inventory_items(tenant_id);
CREATE INDEX idx_inv_item_warehouse ON inventory_items(warehouse_id);
CREATE INDEX idx_inv_item_product ON inventory_items(product_id);
CREATE INDEX idx_inv_item_below_min ON inventory_items(tenant_id) WHERE available_quantity < min_quantity AND min_quantity > 0;
CREATE INDEX idx_inv_item_has_stock ON inventory_items(tenant_id) WHERE available_quantity > 0;

-- Add update trigger for updated_at
CREATE TRIGGER trg_inventory_items_updated_at
    BEFORE UPDATE ON inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create stock_batches table
CREATE TABLE stock_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    inventory_item_id UUID NOT NULL,
    batch_number VARCHAR(50) NOT NULL,
    production_date DATE,
    expiry_date DATE,
    quantity DECIMAL(18, 4) NOT NULL CHECK (quantity >= 0),
    unit_cost DECIMAL(18, 4) NOT NULL CHECK (unit_cost >= 0),
    consumed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key
    CONSTRAINT fk_batch_inventory_item FOREIGN KEY (inventory_item_id) REFERENCES inventory_items(id) ON DELETE CASCADE
);

-- Create indexes for stock_batches
CREATE INDEX idx_batch_inventory_item ON stock_batches(inventory_item_id);
CREATE INDEX idx_batch_number ON stock_batches(inventory_item_id, batch_number);
CREATE INDEX idx_batch_expiry ON stock_batches(expiry_date) WHERE expiry_date IS NOT NULL AND consumed = FALSE;
CREATE INDEX idx_batch_available ON stock_batches(inventory_item_id) WHERE consumed = FALSE AND quantity > 0;

-- Add update trigger for stock_batches
CREATE TRIGGER trg_stock_batches_updated_at
    BEFORE UPDATE ON stock_batches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create stock_locks table
CREATE TABLE stock_locks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    inventory_item_id UUID NOT NULL,
    quantity DECIMAL(18, 4) NOT NULL CHECK (quantity > 0),
    source_type VARCHAR(50) NOT NULL,
    source_id VARCHAR(100) NOT NULL,
    expire_at TIMESTAMPTZ NOT NULL,
    released BOOLEAN NOT NULL DEFAULT FALSE,
    consumed BOOLEAN NOT NULL DEFAULT FALSE,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign key
    CONSTRAINT fk_lock_inventory_item FOREIGN KEY (inventory_item_id) REFERENCES inventory_items(id) ON DELETE CASCADE
);

-- Create indexes for stock_locks
CREATE INDEX idx_lock_inventory_item ON stock_locks(inventory_item_id);
CREATE INDEX idx_lock_src ON stock_locks(source_type, source_id);
CREATE INDEX idx_lock_expire ON stock_locks(expire_at) WHERE released = FALSE AND consumed = FALSE;
CREATE INDEX idx_lock_active ON stock_locks(inventory_item_id) WHERE released = FALSE AND consumed = FALSE;

-- Add update trigger for stock_locks
CREATE TRIGGER trg_stock_locks_updated_at
    BEFORE UPDATE ON stock_locks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add foreign key from inventory_transactions to inventory_items (alter existing table)
ALTER TABLE inventory_transactions
    ADD CONSTRAINT fk_inv_tx_inventory_item
    FOREIGN KEY (inventory_item_id) REFERENCES inventory_items(id) ON DELETE RESTRICT;

ALTER TABLE inventory_transactions
    ADD CONSTRAINT fk_inv_tx_batch
    FOREIGN KEY (batch_id) REFERENCES stock_batches(id) ON DELETE SET NULL;

ALTER TABLE inventory_transactions
    ADD CONSTRAINT fk_inv_tx_lock
    FOREIGN KEY (lock_id) REFERENCES stock_locks(id) ON DELETE SET NULL;

-- Add comments for documentation
COMMENT ON TABLE inventory_items IS 'Inventory items representing stock at warehouse-product level';
COMMENT ON COLUMN inventory_items.available_quantity IS 'Quantity available for sale/use (not locked)';
COMMENT ON COLUMN inventory_items.locked_quantity IS 'Quantity reserved for pending orders';
COMMENT ON COLUMN inventory_items.unit_cost IS 'Moving weighted average cost per unit';
COMMENT ON COLUMN inventory_items.min_quantity IS 'Minimum stock threshold for alerts';
COMMENT ON COLUMN inventory_items.max_quantity IS 'Maximum stock threshold';
COMMENT ON COLUMN inventory_items.version IS 'Version for optimistic locking';

COMMENT ON TABLE stock_batches IS 'Stock batches with production and expiry tracking';
COMMENT ON COLUMN stock_batches.batch_number IS 'Batch/lot number from supplier or production';
COMMENT ON COLUMN stock_batches.consumed IS 'Whether the batch has been fully consumed';

COMMENT ON TABLE stock_locks IS 'Stock reservations for pending orders';
COMMENT ON COLUMN stock_locks.source_type IS 'Type of document that created the lock (e.g., sales_order)';
COMMENT ON COLUMN stock_locks.source_id IS 'ID of the source document';
COMMENT ON COLUMN stock_locks.expire_at IS 'When the lock expires if not consumed';
COMMENT ON COLUMN stock_locks.released IS 'Whether the lock was released (order cancelled)';
COMMENT ON COLUMN stock_locks.consumed IS 'Whether the lock was consumed (order fulfilled)';
