-- Receipt Vouchers table
-- Stores receipt vouchers for payments received from customers

CREATE TABLE IF NOT EXISTS receipt_vouchers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    voucher_number VARCHAR(50) NOT NULL,
    customer_id UUID NOT NULL,
    customer_name VARCHAR(200) NOT NULL,
    amount DECIMAL(18, 4) NOT NULL,
    allocated_amount DECIMAL(18, 4) NOT NULL DEFAULT 0,
    unallocated_amount DECIMAL(18, 4) NOT NULL,
    payment_method VARCHAR(30) NOT NULL,
    payment_reference VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    receipt_date TIMESTAMP NOT NULL,
    remark TEXT,
    confirmed_at TIMESTAMP,
    confirmed_by UUID,
    cancelled_at TIMESTAMP,
    cancelled_by UUID,
    cancel_reason VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- Constraints
    CONSTRAINT chk_receipt_voucher_amount_positive CHECK (amount > 0),
    CONSTRAINT chk_receipt_voucher_allocated_non_negative CHECK (allocated_amount >= 0),
    CONSTRAINT chk_receipt_voucher_unallocated_non_negative CHECK (unallocated_amount >= 0),
    CONSTRAINT chk_receipt_voucher_allocation_valid CHECK (allocated_amount + unallocated_amount = amount),
    CONSTRAINT chk_receipt_voucher_status CHECK (status IN ('DRAFT', 'CONFIRMED', 'ALLOCATED', 'CANCELLED')),
    CONSTRAINT chk_receipt_voucher_payment_method CHECK (payment_method IN ('CASH', 'BANK_TRANSFER', 'WECHAT', 'ALIPAY', 'CHECK', 'BALANCE', 'OTHER'))
);

-- Unique index for voucher number per tenant (only for non-deleted records)
CREATE UNIQUE INDEX idx_receipt_tenant_number ON receipt_vouchers(tenant_id, voucher_number) WHERE deleted_at IS NULL;

-- Foreign key to tenants (if exists)
-- ALTER TABLE receipt_vouchers ADD CONSTRAINT fk_receipt_voucher_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Foreign key to customers (if exists)
-- ALTER TABLE receipt_vouchers ADD CONSTRAINT fk_receipt_voucher_customer FOREIGN KEY (customer_id) REFERENCES customers(id);

-- Indexes for common queries
CREATE INDEX idx_receipt_voucher_tenant ON receipt_vouchers(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_receipt_voucher_customer ON receipt_vouchers(tenant_id, customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_receipt_voucher_status ON receipt_vouchers(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_receipt_voucher_payment_method ON receipt_vouchers(tenant_id, payment_method) WHERE deleted_at IS NULL;
CREATE INDEX idx_receipt_voucher_receipt_date ON receipt_vouchers(tenant_id, receipt_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_receipt_voucher_unallocated ON receipt_vouchers(tenant_id, unallocated_amount) WHERE deleted_at IS NULL AND unallocated_amount > 0;

-- Trigger for updated_at
CREATE TRIGGER update_receipt_vouchers_updated_at
    BEFORE UPDATE ON receipt_vouchers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Receivable Allocations table
-- Stores the allocation of receipt vouchers to account receivables

CREATE TABLE IF NOT EXISTS receivable_allocations (
    id UUID PRIMARY KEY,
    receipt_voucher_id UUID NOT NULL,
    receivable_id UUID NOT NULL,
    receivable_number VARCHAR(50) NOT NULL,
    amount DECIMAL(18, 4) NOT NULL,
    allocated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    remark VARCHAR(500),

    -- Constraints
    CONSTRAINT chk_allocation_amount_positive CHECK (amount > 0),

    -- Foreign keys
    CONSTRAINT fk_allocation_receipt_voucher FOREIGN KEY (receipt_voucher_id) REFERENCES receipt_vouchers(id) ON DELETE CASCADE,
    CONSTRAINT fk_allocation_receivable FOREIGN KEY (receivable_id) REFERENCES account_receivables(id)
);

-- Unique constraint: one voucher can only allocate to one receivable once
CREATE UNIQUE INDEX idx_allocation_voucher_receivable ON receivable_allocations(receipt_voucher_id, receivable_id);

-- Indexes for common queries
CREATE INDEX idx_allocation_voucher ON receivable_allocations(receipt_voucher_id);
CREATE INDEX idx_allocation_receivable ON receivable_allocations(receivable_id);

-- Comments
COMMENT ON TABLE receipt_vouchers IS 'Receipt vouchers for payments received from customers';
COMMENT ON COLUMN receipt_vouchers.voucher_number IS 'Unique voucher number within tenant';
COMMENT ON COLUMN receipt_vouchers.amount IS 'Total receipt amount';
COMMENT ON COLUMN receipt_vouchers.allocated_amount IS 'Amount allocated to receivables';
COMMENT ON COLUMN receipt_vouchers.unallocated_amount IS 'Remaining unallocated amount';
COMMENT ON COLUMN receipt_vouchers.payment_method IS 'Method of payment (CASH, BANK_TRANSFER, WECHAT, ALIPAY, CHECK, BALANCE, OTHER)';
COMMENT ON COLUMN receipt_vouchers.payment_reference IS 'External reference (bank transaction ID, check number, etc.)';
COMMENT ON COLUMN receipt_vouchers.status IS 'Voucher status (DRAFT, CONFIRMED, ALLOCATED, CANCELLED)';
COMMENT ON COLUMN receipt_vouchers.receipt_date IS 'Date when payment was received';

COMMENT ON TABLE receivable_allocations IS 'Allocation of receipt voucher amounts to account receivables';
COMMENT ON COLUMN receivable_allocations.receivable_number IS 'Denormalized receivable number for display';
COMMENT ON COLUMN receivable_allocations.amount IS 'Amount allocated from voucher to receivable';
