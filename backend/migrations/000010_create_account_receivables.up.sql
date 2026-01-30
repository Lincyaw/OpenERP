-- Migration: Create account_receivables and receivable_payment_records tables
-- Description: Creates the core finance tables for tracking amounts owed by customers

-- Create account_receivables table (aggregate root)
CREATE TABLE account_receivables (
    -- Primary key and multi-tenancy
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,

    -- Receivable identification
    receivable_number VARCHAR(50) NOT NULL,

    -- Customer reference
    customer_id UUID NOT NULL,
    customer_name VARCHAR(200) NOT NULL,

    -- Source document reference
    source_type VARCHAR(30) NOT NULL,
    source_id UUID NOT NULL,
    source_number VARCHAR(50) NOT NULL,

    -- Amount tracking (total_amount allows NULL/0 for cancelled/reversed records)
    total_amount DECIMAL(18, 4) CHECK (total_amount IS NULL OR total_amount >= 0),
    paid_amount DECIMAL(18, 4) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    outstanding_amount DECIMAL(18, 4) NOT NULL CHECK (outstanding_amount >= 0),

    -- Status and lifecycle
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    due_date TIMESTAMPTZ,
    remark TEXT,

    -- Payment completion
    paid_at TIMESTAMPTZ,

    -- Reversal tracking
    reversed_at TIMESTAMPTZ,
    reversal_reason VARCHAR(500),

    -- Cancellation tracking
    cancelled_at TIMESTAMPTZ,
    cancel_reason VARCHAR(500),

    -- Timestamps and versioning
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT uq_receivable_tenant_number UNIQUE (tenant_id, receivable_number),
    CONSTRAINT chk_receivable_status CHECK (status IN ('PENDING', 'PARTIAL', 'PAID', 'REVERSED', 'CANCELLED')),
    CONSTRAINT chk_receivable_source_type CHECK (source_type IN ('SALES_ORDER', 'SALES_RETURN', 'MANUAL')),
    -- For cancelled/reversed records: outstanding must be 0
    -- For active records: amounts must balance correctly
    CONSTRAINT chk_receivable_amounts CHECK (
        (status IN ('CANCELLED', 'REVERSED') AND outstanding_amount = 0) OR
        (status NOT IN ('CANCELLED', 'REVERSED') AND paid_amount <= total_amount AND outstanding_amount = total_amount - paid_amount)
    ),

    -- Foreign keys
    CONSTRAINT fk_receivable_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT fk_receivable_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_receivable_tenant ON account_receivables(tenant_id);
CREATE INDEX idx_receivable_customer ON account_receivables(customer_id);
CREATE INDEX idx_receivable_status ON account_receivables(tenant_id, status);
CREATE INDEX idx_receivable_source ON account_receivables(tenant_id, source_type, source_id);
CREATE INDEX idx_receivable_due_date ON account_receivables(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_receivable_outstanding ON account_receivables(tenant_id, outstanding_amount) WHERE status IN ('PENDING', 'PARTIAL');
-- Note: "overdue" records should be calculated at query time using: WHERE due_date < CURRENT_TIMESTAMP
-- Cannot use partial index with NOW() as it's not immutable

-- Add update trigger for updated_at
CREATE TRIGGER trg_account_receivables_updated_at
    BEFORE UPDATE ON account_receivables
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create receivable_payment_records table
CREATE TABLE receivable_payment_records (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Parent receivable
    receivable_id UUID NOT NULL,

    -- Payment voucher reference
    receipt_voucher_id UUID NOT NULL,

    -- Payment details
    amount DECIMAL(18, 4) NOT NULL CHECK (amount > 0),
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remark VARCHAR(500),

    -- Foreign key
    CONSTRAINT fk_payment_record_receivable FOREIGN KEY (receivable_id) REFERENCES account_receivables(id) ON DELETE CASCADE
);

-- Create indexes for payment records
CREATE INDEX idx_payment_record_receivable ON receivable_payment_records(receivable_id);
CREATE INDEX idx_payment_record_voucher ON receivable_payment_records(receipt_voucher_id);
CREATE INDEX idx_payment_record_applied ON receivable_payment_records(applied_at);

-- Add comments for documentation
COMMENT ON TABLE account_receivables IS 'Account receivables tracking amounts owed by customers';
COMMENT ON COLUMN account_receivables.receivable_number IS 'Unique receivable number within tenant (e.g., AR-2024-00001)';
COMMENT ON COLUMN account_receivables.source_type IS 'Type of document that created the receivable (SALES_ORDER, SALES_RETURN, MANUAL)';
COMMENT ON COLUMN account_receivables.source_id IS 'UUID of the source document';
COMMENT ON COLUMN account_receivables.source_number IS 'Human-readable number of the source document';
COMMENT ON COLUMN account_receivables.total_amount IS 'Original amount due';
COMMENT ON COLUMN account_receivables.paid_amount IS 'Amount already paid';
COMMENT ON COLUMN account_receivables.outstanding_amount IS 'Remaining amount due (total - paid)';
COMMENT ON COLUMN account_receivables.status IS 'PENDING=unpaid, PARTIAL=partially paid, PAID=fully paid, REVERSED=voided, CANCELLED=cancelled';
COMMENT ON COLUMN account_receivables.due_date IS 'Payment due date (null = no specific due date)';
COMMENT ON COLUMN account_receivables.version IS 'Version for optimistic locking';

COMMENT ON TABLE receivable_payment_records IS 'Records of payments applied to account receivables';
COMMENT ON COLUMN receivable_payment_records.receipt_voucher_id IS 'Reference to the receipt voucher (will be FK when receipt_vouchers table is created)';
COMMENT ON COLUMN receivable_payment_records.amount IS 'Amount from the voucher applied to this receivable';
