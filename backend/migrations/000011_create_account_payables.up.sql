-- Migration: Create account_payables and payable_payment_records tables
-- Description: Creates the core finance tables for tracking amounts owed to suppliers

-- Create account_payables table (aggregate root)
CREATE TABLE account_payables (
    -- Primary key and multi-tenancy
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,

    -- Payable identification
    payable_number VARCHAR(50) NOT NULL,

    -- Supplier reference
    supplier_id UUID NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,

    -- Source document reference
    source_type VARCHAR(30) NOT NULL,
    source_id UUID NOT NULL,
    source_number VARCHAR(50) NOT NULL,

    -- Amount tracking
    total_amount DECIMAL(18, 4) NOT NULL CHECK (total_amount > 0),
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
    CONSTRAINT uq_payable_tenant_number UNIQUE (tenant_id, payable_number),
    CONSTRAINT chk_payable_status CHECK (status IN ('PENDING', 'PARTIAL', 'PAID', 'REVERSED', 'CANCELLED')),
    CONSTRAINT chk_payable_source_type CHECK (source_type IN ('PURCHASE_ORDER', 'PURCHASE_RETURN', 'MANUAL')),
    CONSTRAINT chk_payable_amounts CHECK (paid_amount <= total_amount AND outstanding_amount = total_amount - paid_amount),

    -- Foreign keys
    CONSTRAINT fk_payable_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT fk_payable_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_payable_tenant ON account_payables(tenant_id);
CREATE INDEX idx_payable_supplier ON account_payables(supplier_id);
CREATE INDEX idx_payable_status ON account_payables(tenant_id, status);
CREATE INDEX idx_payable_source ON account_payables(tenant_id, source_type, source_id);
CREATE INDEX idx_payable_due_date ON account_payables(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_payable_outstanding ON account_payables(tenant_id, outstanding_amount) WHERE status IN ('PENDING', 'PARTIAL');
-- Note: "overdue" records should be calculated at query time using: WHERE due_date < CURRENT_TIMESTAMP
-- Cannot use partial index with NOW() as it's not immutable

-- Add update trigger for updated_at
CREATE TRIGGER trg_account_payables_updated_at
    BEFORE UPDATE ON account_payables
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create payable_payment_records table
CREATE TABLE payable_payment_records (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Parent payable
    payable_id UUID NOT NULL,

    -- Payment voucher reference
    payment_voucher_id UUID NOT NULL,

    -- Payment details
    amount DECIMAL(18, 4) NOT NULL CHECK (amount > 0),
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remark VARCHAR(500),

    -- Foreign key
    CONSTRAINT fk_payment_record_payable FOREIGN KEY (payable_id) REFERENCES account_payables(id) ON DELETE CASCADE
);

-- Create indexes for payment records
CREATE INDEX idx_payable_payment_record_payable ON payable_payment_records(payable_id);
CREATE INDEX idx_payable_payment_record_voucher ON payable_payment_records(payment_voucher_id);
CREATE INDEX idx_payable_payment_record_applied ON payable_payment_records(applied_at);

-- Add comments for documentation
COMMENT ON TABLE account_payables IS 'Account payables tracking amounts owed to suppliers';
COMMENT ON COLUMN account_payables.payable_number IS 'Unique payable number within tenant (e.g., AP-2024-00001)';
COMMENT ON COLUMN account_payables.source_type IS 'Type of document that created the payable (PURCHASE_ORDER, PURCHASE_RETURN, MANUAL)';
COMMENT ON COLUMN account_payables.source_id IS 'UUID of the source document';
COMMENT ON COLUMN account_payables.source_number IS 'Human-readable number of the source document';
COMMENT ON COLUMN account_payables.total_amount IS 'Original amount due';
COMMENT ON COLUMN account_payables.paid_amount IS 'Amount already paid';
COMMENT ON COLUMN account_payables.outstanding_amount IS 'Remaining amount due (total - paid)';
COMMENT ON COLUMN account_payables.status IS 'PENDING=unpaid, PARTIAL=partially paid, PAID=fully paid, REVERSED=voided, CANCELLED=cancelled';
COMMENT ON COLUMN account_payables.due_date IS 'Payment due date (null = no specific due date)';
COMMENT ON COLUMN account_payables.version IS 'Version for optimistic locking';

COMMENT ON TABLE payable_payment_records IS 'Records of payments made for account payables';
COMMENT ON COLUMN payable_payment_records.payment_voucher_id IS 'Reference to the payment voucher (will be FK when payment_vouchers table is created)';
COMMENT ON COLUMN payable_payment_records.amount IS 'Amount from the voucher applied to this payable';
