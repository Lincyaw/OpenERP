-- Migration: 000006_create_suppliers
-- Description: Drop suppliers table

BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS update_suppliers_updated_at ON suppliers;

-- Drop indexes
DROP INDEX IF EXISTS idx_suppliers_over_credit;
DROP INDEX IF EXISTS idx_suppliers_balance;
DROP INDEX IF EXISTS idx_suppliers_code_pattern;
DROP INDEX IF EXISTS idx_suppliers_name_pattern;
DROP INDEX IF EXISTS idx_suppliers_attributes;
DROP INDEX IF EXISTS idx_suppliers_created_at;
DROP INDEX IF EXISTS idx_suppliers_sort_order;
DROP INDEX IF EXISTS idx_suppliers_email;
DROP INDEX IF EXISTS idx_suppliers_phone;
DROP INDEX IF EXISTS idx_suppliers_type;
DROP INDEX IF EXISTS idx_suppliers_status;
DROP INDEX IF EXISTS idx_suppliers_tenant;
DROP INDEX IF EXISTS idx_supplier_tenant_code;

-- Drop table
DROP TABLE IF EXISTS suppliers;

COMMIT;
