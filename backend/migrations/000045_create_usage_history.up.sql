-- Migration: Create usage_history table
-- Description: Stores daily usage snapshots for historical tracking and trend analysis

-- Create usage_history table for storing daily usage snapshots
CREATE TABLE usage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    snapshot_date DATE NOT NULL,
    users_count BIGINT NOT NULL DEFAULT 0,
    products_count BIGINT NOT NULL DEFAULT 0,
    warehouses_count BIGINT NOT NULL DEFAULT 0,
    customers_count BIGINT NOT NULL DEFAULT 0,
    suppliers_count BIGINT NOT NULL DEFAULT 0,
    orders_count BIGINT NOT NULL DEFAULT 0,
    storage_bytes BIGINT NOT NULL DEFAULT 0,
    api_calls_count BIGINT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure one snapshot per tenant per day
    CONSTRAINT uq_usage_history_tenant_date UNIQUE (tenant_id, snapshot_date),

    -- Ensure counts are non-negative
    CONSTRAINT chk_usage_history_users_count CHECK (users_count >= 0),
    CONSTRAINT chk_usage_history_products_count CHECK (products_count >= 0),
    CONSTRAINT chk_usage_history_warehouses_count CHECK (warehouses_count >= 0),
    CONSTRAINT chk_usage_history_customers_count CHECK (customers_count >= 0),
    CONSTRAINT chk_usage_history_suppliers_count CHECK (suppliers_count >= 0),
    CONSTRAINT chk_usage_history_orders_count CHECK (orders_count >= 0),
    CONSTRAINT chk_usage_history_storage_bytes CHECK (storage_bytes >= 0),
    CONSTRAINT chk_usage_history_api_calls_count CHECK (api_calls_count >= 0)
);

-- Create indexes for efficient querying
CREATE INDEX idx_usage_history_tenant_id ON usage_history(tenant_id);
CREATE INDEX idx_usage_history_snapshot_date ON usage_history(snapshot_date);
CREATE INDEX idx_usage_history_tenant_date_range ON usage_history(tenant_id, snapshot_date DESC);

-- Create partial index for recent data (last 90 days) for faster queries
CREATE INDEX idx_usage_history_recent ON usage_history(tenant_id, snapshot_date)
    WHERE snapshot_date > CURRENT_DATE - INTERVAL '90 days';

-- Add comments for documentation
COMMENT ON TABLE usage_history IS 'Daily usage snapshots for historical tracking and trend analysis';
COMMENT ON COLUMN usage_history.tenant_id IS 'The tenant this snapshot belongs to';
COMMENT ON COLUMN usage_history.snapshot_date IS 'The date of this usage snapshot';
COMMENT ON COLUMN usage_history.users_count IS 'Number of active users at snapshot time';
COMMENT ON COLUMN usage_history.products_count IS 'Number of products/SKUs at snapshot time';
COMMENT ON COLUMN usage_history.warehouses_count IS 'Number of warehouses at snapshot time';
COMMENT ON COLUMN usage_history.customers_count IS 'Number of customers at snapshot time';
COMMENT ON COLUMN usage_history.suppliers_count IS 'Number of suppliers at snapshot time';
COMMENT ON COLUMN usage_history.orders_count IS 'Cumulative orders created up to snapshot date';
COMMENT ON COLUMN usage_history.storage_bytes IS 'Total storage used in bytes at snapshot time';
COMMENT ON COLUMN usage_history.api_calls_count IS 'API calls made on the snapshot date';
COMMENT ON COLUMN usage_history.metadata IS 'Additional metrics and context as JSON';
