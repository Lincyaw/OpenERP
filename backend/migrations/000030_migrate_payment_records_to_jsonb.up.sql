-- Migration: Migrate receivable_payment_records from separate table to JSONB column
-- Description: This migration moves PaymentRecord data from a separate table into a JSONB column
--              on the account_receivables table, following DDD aggregate root persistence pattern.

-- Step 1: Add the payment_records JSONB column to account_receivables
ALTER TABLE account_receivables
ADD COLUMN IF NOT EXISTS payment_records JSONB DEFAULT '[]'::JSONB;

-- Step 2: Migrate existing data from receivable_payment_records to the JSONB column
UPDATE account_receivables ar
SET payment_records = COALESCE(
    (
        SELECT jsonb_agg(
            jsonb_build_object(
                'id', rpr.id,
                'receipt_voucher_id', rpr.receipt_voucher_id,
                'amount', rpr.amount::text,
                'applied_at', rpr.applied_at,
                'remark', COALESCE(rpr.remark, '')
            )
            ORDER BY rpr.applied_at ASC
        )
        FROM receivable_payment_records rpr
        WHERE rpr.receivable_id = ar.id
    ),
    '[]'::JSONB
);

-- Step 3: Drop the old receivable_payment_records table
DROP TABLE IF EXISTS receivable_payment_records;

-- Step 4: Add index for JSONB querying (optional, for performance)
-- This creates a GIN index that can speed up JSONB containment queries
CREATE INDEX IF NOT EXISTS idx_receivable_payment_records_gin
ON account_receivables USING GIN (payment_records);

-- Add comment for documentation
COMMENT ON COLUMN account_receivables.payment_records IS 'JSONB array of payment records applied to this receivable, stored within aggregate boundary';
