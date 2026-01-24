-- Rollback migration: Drop account_payables and payable_payment_records tables

-- Drop child table first (payment records)
DROP TABLE IF EXISTS payable_payment_records;

-- Drop trigger before dropping table
DROP TRIGGER IF EXISTS trg_account_payables_updated_at ON account_payables;

-- Drop main table
DROP TABLE IF EXISTS account_payables;
