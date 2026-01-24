-- Migration: Drop balance_transactions table
-- Description: Removes the balance_transactions table

DROP TRIGGER IF EXISTS update_balance_transactions_updated_at ON balance_transactions;
DROP TABLE IF EXISTS balance_transactions;
