-- Down migration: Restore original constraints
-- Note: This may fail if there are records that violate the original constraints

-- Restore account_receivables constraints
ALTER TABLE account_receivables DROP CONSTRAINT IF EXISTS chk_receivable_amounts;
ALTER TABLE account_receivables DROP CONSTRAINT IF EXISTS chk_receivable_total_positive;

ALTER TABLE account_receivables
ALTER COLUMN total_amount SET NOT NULL;
ALTER TABLE account_receivables
ADD CONSTRAINT account_receivables_total_amount_check CHECK (total_amount > 0);
ALTER TABLE account_receivables
ADD CONSTRAINT chk_receivable_amounts CHECK (paid_amount <= total_amount AND outstanding_amount = total_amount - paid_amount);

-- Restore account_payables constraints
ALTER TABLE account_payables DROP CONSTRAINT IF EXISTS chk_payable_amounts;
ALTER TABLE account_payables DROP CONSTRAINT IF EXISTS chk_payable_total_positive;

ALTER TABLE account_payables
ALTER COLUMN total_amount SET NOT NULL;
ALTER TABLE account_payables
ADD CONSTRAINT account_payables_total_amount_check CHECK (total_amount > 0);
ALTER TABLE account_payables
ADD CONSTRAINT chk_payable_amounts CHECK (paid_amount <= total_amount AND outstanding_amount = total_amount - paid_amount);
