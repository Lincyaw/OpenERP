-- Migration: Create expense_records and other_income_records tables
-- Description: Creates tables for tracking non-trade expenses and income

-- Create expense_records table (aggregate root)
CREATE TABLE expense_records (
    -- Primary key and multi-tenancy
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,

    -- Expense identification
    expense_number VARCHAR(50) NOT NULL,

    -- Expense details
    category VARCHAR(30) NOT NULL,
    amount DECIMAL(18, 4) NOT NULL CHECK (amount > 0),
    description VARCHAR(500) NOT NULL,
    incurred_at TIMESTAMPTZ NOT NULL,

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    payment_status VARCHAR(20) NOT NULL DEFAULT 'UNPAID',
    payment_method VARCHAR(30),
    paid_at TIMESTAMPTZ,

    -- Additional info
    remark TEXT,
    attachment_urls TEXT, -- JSON array of attachment URLs

    -- Submission tracking
    submitted_at TIMESTAMPTZ,
    submitted_by UUID,

    -- Approval tracking
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    approval_remark VARCHAR(500),

    -- Rejection tracking
    rejected_at TIMESTAMPTZ,
    rejected_by UUID,
    rejection_reason VARCHAR(500),

    -- Cancellation tracking
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID,
    cancel_reason VARCHAR(500),

    -- Timestamps and versioning
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT uq_expense_tenant_number UNIQUE (tenant_id, expense_number),
    CONSTRAINT chk_expense_category CHECK (category IN (
        'RENT', 'UTILITIES', 'SALARY', 'OFFICE', 'TRAVEL',
        'MARKETING', 'EQUIPMENT', 'MAINTENANCE', 'INSURANCE', 'TAX', 'OTHER'
    )),
    CONSTRAINT chk_expense_status CHECK (status IN ('DRAFT', 'PENDING', 'APPROVED', 'REJECTED', 'CANCELLED')),
    CONSTRAINT chk_expense_payment_status CHECK (payment_status IN ('UNPAID', 'PAID')),
    CONSTRAINT chk_expense_payment_method CHECK (payment_method IS NULL OR payment_method IN (
        'CASH', 'BANK_TRANSFER', 'WECHAT_PAY', 'ALIPAY', 'CHECK', 'CREDIT_CARD', 'BALANCE', 'OTHER'
    )),

    -- Foreign keys
    CONSTRAINT fk_expense_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_expense_tenant ON expense_records(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_expense_category ON expense_records(tenant_id, category) WHERE deleted_at IS NULL;
CREATE INDEX idx_expense_status ON expense_records(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_expense_payment_status ON expense_records(tenant_id, payment_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_expense_incurred ON expense_records(tenant_id, incurred_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_expense_pending ON expense_records(tenant_id) WHERE status = 'PENDING' AND deleted_at IS NULL;
CREATE INDEX idx_expense_deleted ON expense_records(deleted_at) WHERE deleted_at IS NOT NULL;

-- Add update trigger for updated_at
CREATE TRIGGER trg_expense_records_updated_at
    BEFORE UPDATE ON expense_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create other_income_records table (aggregate root)
CREATE TABLE other_income_records (
    -- Primary key and multi-tenancy
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,

    -- Income identification
    income_number VARCHAR(50) NOT NULL,

    -- Income details
    category VARCHAR(30) NOT NULL,
    amount DECIMAL(18, 4) NOT NULL CHECK (amount > 0),
    description VARCHAR(500) NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    receipt_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    payment_method VARCHAR(30),
    actual_received TIMESTAMPTZ,

    -- Additional info
    remark TEXT,
    attachment_urls TEXT, -- JSON array of attachment URLs

    -- Confirmation tracking
    confirmed_at TIMESTAMPTZ,
    confirmed_by UUID,

    -- Cancellation tracking
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID,
    cancel_reason VARCHAR(500),

    -- Timestamps and versioning
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Constraints
    CONSTRAINT uq_income_tenant_number UNIQUE (tenant_id, income_number),
    CONSTRAINT chk_income_category CHECK (category IN (
        'INVESTMENT', 'SUBSIDY', 'INTEREST', 'RENTAL',
        'REFUND', 'COMPENSATION', 'ASSET_DISPOSAL', 'OTHER'
    )),
    CONSTRAINT chk_income_status CHECK (status IN ('DRAFT', 'CONFIRMED', 'CANCELLED')),
    CONSTRAINT chk_income_receipt_status CHECK (receipt_status IN ('PENDING', 'RECEIVED')),
    CONSTRAINT chk_income_payment_method CHECK (payment_method IS NULL OR payment_method IN (
        'CASH', 'BANK_TRANSFER', 'WECHAT_PAY', 'ALIPAY', 'CHECK', 'CREDIT_CARD', 'BALANCE', 'OTHER'
    )),

    -- Foreign keys
    CONSTRAINT fk_income_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_income_tenant ON other_income_records(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_income_category ON other_income_records(tenant_id, category) WHERE deleted_at IS NULL;
CREATE INDEX idx_income_status ON other_income_records(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_income_receipt_status ON other_income_records(tenant_id, receipt_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_income_received ON other_income_records(tenant_id, received_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_income_draft ON other_income_records(tenant_id) WHERE status = 'DRAFT' AND deleted_at IS NULL;
CREATE INDEX idx_income_deleted ON other_income_records(deleted_at) WHERE deleted_at IS NOT NULL;

-- Add update trigger for updated_at
CREATE TRIGGER trg_other_income_records_updated_at
    BEFORE UPDATE ON other_income_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE expense_records IS 'Non-trade expense records (rent, utilities, salary, etc.)';
COMMENT ON COLUMN expense_records.expense_number IS 'Unique expense number within tenant (e.g., EXP-2024-00001)';
COMMENT ON COLUMN expense_records.category IS 'Expense category (RENT, UTILITIES, SALARY, OFFICE, etc.)';
COMMENT ON COLUMN expense_records.description IS 'Description of the expense';
COMMENT ON COLUMN expense_records.incurred_at IS 'Date when the expense was incurred';
COMMENT ON COLUMN expense_records.status IS 'Approval status (DRAFT, PENDING, APPROVED, REJECTED, CANCELLED)';
COMMENT ON COLUMN expense_records.payment_status IS 'Payment status (UNPAID, PAID)';
COMMENT ON COLUMN expense_records.attachment_urls IS 'JSON array of attachment URLs for receipts/invoices';
COMMENT ON COLUMN expense_records.version IS 'Version for optimistic locking';

COMMENT ON TABLE other_income_records IS 'Non-trade income records (investment returns, subsidies, interest, etc.)';
COMMENT ON COLUMN other_income_records.income_number IS 'Unique income number within tenant (e.g., INC-2024-00001)';
COMMENT ON COLUMN other_income_records.category IS 'Income category (INVESTMENT, SUBSIDY, INTEREST, RENTAL, etc.)';
COMMENT ON COLUMN other_income_records.description IS 'Description of the income';
COMMENT ON COLUMN other_income_records.received_at IS 'Date when the income was received';
COMMENT ON COLUMN other_income_records.status IS 'Confirmation status (DRAFT, CONFIRMED, CANCELLED)';
COMMENT ON COLUMN other_income_records.receipt_status IS 'Receipt status (PENDING, RECEIVED)';
COMMENT ON COLUMN other_income_records.attachment_urls IS 'JSON array of attachment URLs for supporting documents';
COMMENT ON COLUMN other_income_records.version IS 'Version for optimistic locking';
