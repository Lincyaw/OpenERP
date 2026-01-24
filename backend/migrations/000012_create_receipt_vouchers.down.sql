-- Drop receivable allocations first (child table)
DROP TABLE IF EXISTS receivable_allocations;

-- Drop trigger
DROP TRIGGER IF EXISTS update_receipt_vouchers_updated_at ON receipt_vouchers;

-- Drop receipt vouchers table
DROP TABLE IF EXISTS receipt_vouchers;
