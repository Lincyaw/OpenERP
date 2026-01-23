-- Migration: Drop customers table
-- Description: Rollback migration for customers table

-- Drop trigger first
DROP TRIGGER IF EXISTS update_customers_updated_at ON customers;

-- Drop indexes
DROP INDEX IF EXISTS idx_customer_tenant_id;
DROP INDEX IF EXISTS idx_customer_tenant_code;
DROP INDEX IF EXISTS idx_customer_tenant_status;
DROP INDEX IF EXISTS idx_customer_tenant_type;
DROP INDEX IF EXISTS idx_customer_tenant_level;
DROP INDEX IF EXISTS idx_customer_tenant_phone;
DROP INDEX IF EXISTS idx_customer_tenant_email;
DROP INDEX IF EXISTS idx_customer_tenant_sort;
DROP INDEX IF EXISTS idx_customer_tenant_balance;
DROP INDEX IF EXISTS idx_customer_name_pattern;
DROP INDEX IF EXISTS idx_customer_code_pattern;
DROP INDEX IF EXISTS idx_customer_attributes;

-- Drop table
DROP TABLE IF EXISTS customers;
