-- Migration: Drop expense_records and other_income_records tables
-- Description: Rollback migration 000014

-- Drop triggers
DROP TRIGGER IF EXISTS trg_other_income_records_updated_at ON other_income_records;
DROP TRIGGER IF EXISTS trg_expense_records_updated_at ON expense_records;

-- Drop indexes for other_income_records
DROP INDEX IF EXISTS idx_income_deleted;
DROP INDEX IF EXISTS idx_income_draft;
DROP INDEX IF EXISTS idx_income_received;
DROP INDEX IF EXISTS idx_income_receipt_status;
DROP INDEX IF EXISTS idx_income_status;
DROP INDEX IF EXISTS idx_income_category;
DROP INDEX IF EXISTS idx_income_tenant;

-- Drop indexes for expense_records
DROP INDEX IF EXISTS idx_expense_deleted;
DROP INDEX IF EXISTS idx_expense_pending;
DROP INDEX IF EXISTS idx_expense_incurred;
DROP INDEX IF EXISTS idx_expense_payment_status;
DROP INDEX IF EXISTS idx_expense_status;
DROP INDEX IF EXISTS idx_expense_category;
DROP INDEX IF EXISTS idx_expense_tenant;

-- Drop tables
DROP TABLE IF EXISTS other_income_records;
DROP TABLE IF EXISTS expense_records;
