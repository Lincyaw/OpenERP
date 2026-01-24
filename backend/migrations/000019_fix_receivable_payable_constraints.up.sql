-- Migration: Fix account_receivables and account_payables constraints
-- Description: Modify amount constraints to allow cancelled/reversed records to have zero outstanding

-- Drop old constraints and add new ones for account_receivables
ALTER TABLE account_receivables DROP CONSTRAINT IF EXISTS chk_receivable_amounts;
ALTER TABLE account_receivables DROP CONSTRAINT IF EXISTS account_receivables_total_amount_check;

-- Add constraint that allows cancelled/reversed records to have zero outstanding even when total > 0
-- For active records (PENDING, PARTIAL, PAID): outstanding_amount = total_amount - paid_amount
-- For terminal records (CANCELLED, REVERSED): outstanding_amount can be 0 regardless of amounts
ALTER TABLE account_receivables
ADD CONSTRAINT chk_receivable_amounts CHECK (
    (status IN ('CANCELLED', 'REVERSED') AND outstanding_amount = 0) OR
    (status NOT IN ('CANCELLED', 'REVERSED') AND paid_amount <= total_amount AND outstanding_amount = total_amount - paid_amount)
);

-- Allow total_amount to be > 0 (required for new records) but allow historical reference
-- Drop the existing check and recreate with >= 0 for flexibility
ALTER TABLE account_receivables
ALTER COLUMN total_amount DROP NOT NULL;
ALTER TABLE account_receivables
ADD CONSTRAINT chk_receivable_total_positive CHECK (total_amount IS NULL OR total_amount >= 0);

-- Similarly fix account_payables constraints
ALTER TABLE account_payables DROP CONSTRAINT IF EXISTS chk_payable_amounts;
ALTER TABLE account_payables DROP CONSTRAINT IF EXISTS account_payables_total_amount_check;

ALTER TABLE account_payables
ADD CONSTRAINT chk_payable_amounts CHECK (
    (status IN ('CANCELLED', 'REVERSED') AND outstanding_amount = 0) OR
    (status NOT IN ('CANCELLED', 'REVERSED') AND paid_amount <= total_amount AND outstanding_amount = total_amount - paid_amount)
);

ALTER TABLE account_payables
ALTER COLUMN total_amount DROP NOT NULL;
ALTER TABLE account_payables
ADD CONSTRAINT chk_payable_total_positive CHECK (total_amount IS NULL OR total_amount >= 0);
