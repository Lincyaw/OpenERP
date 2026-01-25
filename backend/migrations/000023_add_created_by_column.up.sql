-- Migration: add_created_by_column
-- Created: 2026-01-25
-- Description: Add created_by column to all tables using TenantAggregateRoot for data scope filtering

-- Products module
ALTER TABLE products ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_products_created_by ON products(created_by);
COMMENT ON COLUMN products.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE categories ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_categories_created_by ON categories(created_by);
COMMENT ON COLUMN categories.created_by IS 'User who created this record (for data scope filtering)';

-- Partner module
ALTER TABLE customers ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_customers_created_by ON customers(created_by);
COMMENT ON COLUMN customers.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_suppliers_created_by ON suppliers(created_by);
COMMENT ON COLUMN suppliers.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE warehouses ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_warehouses_created_by ON warehouses(created_by);
COMMENT ON COLUMN warehouses.created_by IS 'User who created this record (for data scope filtering)';

-- Inventory module
ALTER TABLE inventory_items ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_inventory_items_created_by ON inventory_items(created_by);
COMMENT ON COLUMN inventory_items.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE stock_takings ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_stock_takings_created_by ON stock_takings(created_by);
COMMENT ON COLUMN stock_takings.created_by IS 'User who created this record (for data scope filtering)';

-- Trade module
ALTER TABLE sales_orders ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_sales_orders_created_by ON sales_orders(created_by);
COMMENT ON COLUMN sales_orders.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE purchase_orders ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_purchase_orders_created_by ON purchase_orders(created_by);
COMMENT ON COLUMN purchase_orders.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE sales_returns ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_sales_returns_created_by ON sales_returns(created_by);
COMMENT ON COLUMN sales_returns.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE purchase_returns ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_purchase_returns_created_by ON purchase_returns(created_by);
COMMENT ON COLUMN purchase_returns.created_by IS 'User who created this record (for data scope filtering)';

-- Finance module
ALTER TABLE account_receivables ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_account_receivables_created_by ON account_receivables(created_by);
COMMENT ON COLUMN account_receivables.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE account_payables ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_account_payables_created_by ON account_payables(created_by);
COMMENT ON COLUMN account_payables.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE payment_vouchers ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_payment_vouchers_created_by ON payment_vouchers(created_by);
COMMENT ON COLUMN payment_vouchers.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE receipt_vouchers ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_receipt_vouchers_created_by ON receipt_vouchers(created_by);
COMMENT ON COLUMN receipt_vouchers.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE expense_records ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_expense_records_created_by ON expense_records(created_by);
COMMENT ON COLUMN expense_records.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE other_income_records ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_other_income_records_created_by ON other_income_records(created_by);
COMMENT ON COLUMN other_income_records.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE debit_memos ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_debit_memos_created_by ON debit_memos(created_by);
COMMENT ON COLUMN debit_memos.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE credit_memos ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_credit_memos_created_by ON credit_memos(created_by);
COMMENT ON COLUMN credit_memos.created_by IS 'User who created this record (for data scope filtering)';

-- Identity module
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_users_created_by ON users(created_by);
COMMENT ON COLUMN users.created_by IS 'User who created this record (for data scope filtering)';

ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_by UUID;
CREATE INDEX IF NOT EXISTS idx_roles_created_by ON roles(created_by);
COMMENT ON COLUMN roles.created_by IS 'User who created this record (for data scope filtering)';
