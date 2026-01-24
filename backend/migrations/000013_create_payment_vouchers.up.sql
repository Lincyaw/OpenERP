-- Payment Vouchers table
-- Stores payment vouchers for payments made to suppliers

CREATE TABLE IF NOT EXISTS payment_vouchers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    voucher_number VARCHAR(50) NOT NULL,
    supplier_id UUID NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,
    amount DECIMAL(18, 4) NOT NULL,
    allocated_amount DECIMAL(18, 4) NOT NULL DEFAULT 0,
    unallocated_amount DECIMAL(18, 4) NOT NULL,
    payment_method VARCHAR(30) NOT NULL,
    payment_reference VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    payment_date TIMESTAMP NOT NULL,
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
    CONSTRAINT chk_payment_voucher_amount_positive CHECK (amount > 0),
    CONSTRAINT chk_payment_voucher_allocated_non_negative CHECK (allocated_amount >= 0),
    CONSTRAINT chk_payment_voucher_unallocated_non_negative CHECK (unallocated_amount >= 0),
    CONSTRAINT chk_payment_voucher_allocation_valid CHECK (allocated_amount + unallocated_amount = amount),
    CONSTRAINT chk_payment_voucher_status CHECK (status IN ('DRAFT', 'CONFIRMED', 'ALLOCATED', 'CANCELLED')),
    CONSTRAINT chk_payment_voucher_payment_method CHECK (payment_method IN ('CASH', 'BANK_TRANSFER', 'WECHAT', 'ALIPAY', 'CHECK', 'BALANCE', 'OTHER'))
);

-- Unique index for voucher number per tenant (only for non-deleted records)
CREATE UNIQUE INDEX idx_payment_tenant_number ON payment_vouchers(tenant_id, voucher_number) WHERE deleted_at IS NULL;

-- Foreign key to tenants (if exists)
-- ALTER TABLE payment_vouchers ADD CONSTRAINT fk_payment_voucher_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Foreign key to suppliers (if exists)
-- ALTER TABLE payment_vouchers ADD CONSTRAINT fk_payment_voucher_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id);

-- Indexes for common queries
CREATE INDEX idx_payment_voucher_tenant ON payment_vouchers(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_voucher_supplier ON payment_vouchers(tenant_id, supplier_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_voucher_status ON payment_vouchers(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_voucher_payment_method ON payment_vouchers(tenant_id, payment_method) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_voucher_payment_date ON payment_vouchers(tenant_id, payment_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_voucher_unallocated ON payment_vouchers(tenant_id, unallocated_amount) WHERE deleted_at IS NULL AND unallocated_amount > 0;

-- Trigger for updated_at
CREATE TRIGGER update_payment_vouchers_updated_at
    BEFORE UPDATE ON payment_vouchers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Payable Allocations table
-- Stores the allocation of payment vouchers to account payables

CREATE TABLE IF NOT EXISTS payable_allocations (
    id UUID PRIMARY KEY,
    payment_voucher_id UUID NOT NULL,
    payable_id UUID NOT NULL,
    payable_number VARCHAR(50) NOT NULL,
    amount DECIMAL(18, 4) NOT NULL,
    allocated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    remark VARCHAR(500),

    -- Constraints
    CONSTRAINT chk_payable_allocation_amount_positive CHECK (amount > 0),

    -- Foreign keys
    CONSTRAINT fk_payable_allocation_payment_voucher FOREIGN KEY (payment_voucher_id) REFERENCES payment_vouchers(id) ON DELETE CASCADE,
    CONSTRAINT fk_payable_allocation_payable FOREIGN KEY (payable_id) REFERENCES account_payables(id)
);

-- Unique constraint: one voucher can only allocate to one payable once
CREATE UNIQUE INDEX idx_payable_allocation_voucher_payable ON payable_allocations(payment_voucher_id, payable_id);

-- Indexes for common queries
CREATE INDEX idx_payable_allocation_voucher ON payable_allocations(payment_voucher_id);
CREATE INDEX idx_payable_allocation_payable ON payable_allocations(payable_id);

-- Comments
COMMENT ON TABLE payment_vouchers IS 'Payment vouchers for payments made to suppliers';
COMMENT ON COLUMN payment_vouchers.voucher_number IS 'Unique voucher number within tenant';
COMMENT ON COLUMN payment_vouchers.amount IS 'Total payment amount';
COMMENT ON COLUMN payment_vouchers.allocated_amount IS 'Amount allocated to payables';
COMMENT ON COLUMN payment_vouchers.unallocated_amount IS 'Remaining unallocated amount';
COMMENT ON COLUMN payment_vouchers.payment_method IS 'Method of payment (CASH, BANK_TRANSFER, WECHAT, ALIPAY, CHECK, BALANCE, OTHER)';
COMMENT ON COLUMN payment_vouchers.payment_reference IS 'External reference (bank transaction ID, check number, etc.)';
COMMENT ON COLUMN payment_vouchers.status IS 'Voucher status (DRAFT, CONFIRMED, ALLOCATED, CANCELLED)';
COMMENT ON COLUMN payment_vouchers.payment_date IS 'Date when payment was made';

COMMENT ON TABLE payable_allocations IS 'Allocation of payment voucher amounts to account payables';
COMMENT ON COLUMN payable_allocations.payable_number IS 'Denormalized payable number for display';
COMMENT ON COLUMN payable_allocations.amount IS 'Amount allocated from voucher to payable';
