-- Down migration: Drop account_receivables and receivable_payment_records tables

-- Drop payment records table first (child table)
DROP TABLE IF EXISTS receivable_payment_records;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_account_receivables_updated_at ON account_receivables;

-- Drop main table
DROP TABLE IF EXISTS account_receivables;
