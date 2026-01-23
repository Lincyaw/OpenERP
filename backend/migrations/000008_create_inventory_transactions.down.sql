-- Rollback: Drop inventory_transactions table and related objects

-- Drop triggers
DROP TRIGGER IF EXISTS trg_prevent_inv_tx_delete ON inventory_transactions;
DROP TRIGGER IF EXISTS trg_prevent_inv_tx_update ON inventory_transactions;

-- Drop trigger functions
DROP FUNCTION IF EXISTS prevent_inventory_transaction_delete();
DROP FUNCTION IF EXISTS prevent_inventory_transaction_update();

-- Drop table
DROP TABLE IF EXISTS inventory_transactions;

-- Drop enums
DROP TYPE IF EXISTS source_type;
DROP TYPE IF EXISTS transaction_type;
