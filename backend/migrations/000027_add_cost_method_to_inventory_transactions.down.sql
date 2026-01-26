-- Remove cost_method column from inventory_transactions table

ALTER TABLE inventory_transactions
DROP COLUMN IF EXISTS cost_method;
