-- Migration rollback: add_created_by_column
-- Description: Remove created_by column from all tables

-- Products module
DROP INDEX IF EXISTS idx_products_created_by;
ALTER TABLE products DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_categories_created_by;
ALTER TABLE categories DROP COLUMN IF EXISTS created_by;

-- Partner module
DROP INDEX IF EXISTS idx_customers_created_by;
ALTER TABLE customers DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_suppliers_created_by;
ALTER TABLE suppliers DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_warehouses_created_by;
ALTER TABLE warehouses DROP COLUMN IF EXISTS created_by;

-- Inventory module
DROP INDEX IF EXISTS idx_inventory_items_created_by;
ALTER TABLE inventory_items DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_stock_takings_created_by;
ALTER TABLE stock_takings DROP COLUMN IF EXISTS created_by;

-- Trade module
DROP INDEX IF EXISTS idx_sales_orders_created_by;
ALTER TABLE sales_orders DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_purchase_orders_created_by;
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_sales_returns_created_by;
ALTER TABLE sales_returns DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_purchase_returns_created_by;
ALTER TABLE purchase_returns DROP COLUMN IF EXISTS created_by;

-- Finance module
DROP INDEX IF EXISTS idx_account_receivables_created_by;
ALTER TABLE account_receivables DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_account_payables_created_by;
ALTER TABLE account_payables DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_payment_vouchers_created_by;
ALTER TABLE payment_vouchers DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_receipt_vouchers_created_by;
ALTER TABLE receipt_vouchers DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_expense_records_created_by;
ALTER TABLE expense_records DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_other_income_records_created_by;
ALTER TABLE other_income_records DROP COLUMN IF EXISTS created_by;

-- Note: debit_memos and credit_memos tables are not yet created, skipping these for now

-- Identity module
DROP INDEX IF EXISTS idx_users_created_by;
ALTER TABLE users DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_roles_created_by;
ALTER TABLE roles DROP COLUMN IF EXISTS created_by;
