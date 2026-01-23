-- Rollback: Drop inventory_items, stock_batches, and stock_locks tables

-- Remove foreign keys from inventory_transactions first
ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS fk_inv_tx_lock;
ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS fk_inv_tx_batch;
ALTER TABLE inventory_transactions DROP CONSTRAINT IF EXISTS fk_inv_tx_inventory_item;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_stock_locks_updated_at ON stock_locks;
DROP TRIGGER IF EXISTS trg_stock_batches_updated_at ON stock_batches;
DROP TRIGGER IF EXISTS trg_inventory_items_updated_at ON inventory_items;

-- Drop tables in reverse order (respecting foreign key dependencies)
DROP TABLE IF EXISTS stock_locks;
DROP TABLE IF EXISTS stock_batches;
DROP TABLE IF EXISTS inventory_items;
