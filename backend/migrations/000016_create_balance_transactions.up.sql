-- Migration: Create balance_transactions table
-- Description: Creates the balance_transactions table for tracking customer balance changes

CREATE TABLE balance_transactions (
    -- Primary key
    id UUID PRIMARY KEY,

    -- Multi-tenancy
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Customer reference
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,

    -- Transaction details
    transaction_type VARCHAR(30) NOT NULL,
    amount DECIMAL(18,4) NOT NULL,
    balance_before DECIMAL(18,4) NOT NULL,
    balance_after DECIMAL(18,4) NOT NULL,

    -- Source document reference
    source_type VARCHAR(30) NOT NULL,
    source_id VARCHAR(50),

    -- Additional information
    reference VARCHAR(100),
    remark VARCHAR(500),
    operator_id UUID,

    -- Transaction timestamp
    transaction_date TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Entity timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT chk_balance_tx_type CHECK (transaction_type IN ('RECHARGE', 'CONSUME', 'REFUND', 'ADJUSTMENT', 'EXPIRE')),
    CONSTRAINT chk_balance_tx_source_type CHECK (source_type IN ('MANUAL', 'SALES_ORDER', 'SALES_RETURN', 'RECEIPT_VOUCHER', 'SYSTEM')),
    CONSTRAINT chk_balance_tx_amount CHECK (amount > 0),
    CONSTRAINT chk_balance_tx_balance_before CHECK (balance_before >= 0),
    CONSTRAINT chk_balance_tx_balance_after CHECK (balance_after >= 0)
);

-- Indexes for tenant and time-based queries (composite for efficiency)
CREATE INDEX idx_bal_tx_tenant_time ON balance_transactions(tenant_id, transaction_date DESC);

-- Index for customer-specific queries
CREATE INDEX idx_bal_tx_customer ON balance_transactions(customer_id);
CREATE INDEX idx_bal_tx_tenant_customer ON balance_transactions(tenant_id, customer_id, transaction_date DESC);

-- Index for transaction type filtering
CREATE INDEX idx_bal_tx_type ON balance_transactions(transaction_type);

-- Index for source document lookup
CREATE INDEX idx_bal_tx_source ON balance_transactions(source_type, source_id) WHERE source_id IS NOT NULL;

-- Auto-update updated_at trigger
CREATE TRIGGER update_balance_transactions_updated_at
    BEFORE UPDATE ON balance_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE balance_transactions IS 'Immutable ledger of all customer balance changes';
COMMENT ON COLUMN balance_transactions.transaction_type IS 'Type: RECHARGE, CONSUME, REFUND, ADJUSTMENT, EXPIRE';
COMMENT ON COLUMN balance_transactions.amount IS 'Transaction amount (always positive, direction determined by type)';
COMMENT ON COLUMN balance_transactions.balance_before IS 'Customer balance before this transaction';
COMMENT ON COLUMN balance_transactions.balance_after IS 'Customer balance after this transaction';
COMMENT ON COLUMN balance_transactions.source_type IS 'Source: MANUAL, SALES_ORDER, SALES_RETURN, RECEIPT_VOUCHER, SYSTEM';
COMMENT ON COLUMN balance_transactions.source_id IS 'ID of the source document (optional)';
COMMENT ON COLUMN balance_transactions.reference IS 'Reference number for the transaction';
COMMENT ON COLUMN balance_transactions.remark IS 'Additional notes or remarks';
COMMENT ON COLUMN balance_transactions.operator_id IS 'User who performed the operation';
COMMENT ON COLUMN balance_transactions.transaction_date IS 'When the transaction occurred';
