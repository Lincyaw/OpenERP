-- Rollback migration: Restore receivable_payment_records table from JSONB column
-- Description: This rollback recreates the separate table and migrates data back from JSONB

-- Step 1: Recreate the receivable_payment_records table
CREATE TABLE IF NOT EXISTS receivable_payment_records (
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

-- Step 2: Create indexes for payment records
CREATE INDEX IF NOT EXISTS idx_payment_record_receivable ON receivable_payment_records(receivable_id);
CREATE INDEX IF NOT EXISTS idx_payment_record_voucher ON receivable_payment_records(receipt_voucher_id);
CREATE INDEX IF NOT EXISTS idx_payment_record_applied ON receivable_payment_records(applied_at);

-- Step 3: Migrate data back from JSONB to the table
INSERT INTO receivable_payment_records (id, receivable_id, receipt_voucher_id, amount, applied_at, remark)
SELECT
    (record->>'id')::UUID,
    ar.id,
    (record->>'receipt_voucher_id')::UUID,
    (record->>'amount')::DECIMAL(18, 4),
    (record->>'applied_at')::TIMESTAMPTZ,
    NULLIF(record->>'remark', '')
FROM account_receivables ar,
     jsonb_array_elements(ar.payment_records) AS record
WHERE jsonb_array_length(ar.payment_records) > 0;

-- Step 4: Drop the JSONB column and GIN index
DROP INDEX IF EXISTS idx_receivable_payment_records_gin;
ALTER TABLE account_receivables DROP COLUMN IF EXISTS payment_records;

-- Add comment for documentation
COMMENT ON TABLE receivable_payment_records IS 'Records of payments applied to account receivables';
COMMENT ON COLUMN receivable_payment_records.receipt_voucher_id IS 'Reference to the receipt voucher';
COMMENT ON COLUMN receivable_payment_records.amount IS 'Amount from the voucher applied to this receivable';
