-- Drop payable allocations first (child table)
DROP TABLE IF EXISTS payable_allocations;

-- Drop trigger
DROP TRIGGER IF EXISTS update_payment_vouchers_updated_at ON payment_vouchers;

-- Drop payment vouchers table
DROP TABLE IF EXISTS payment_vouchers;
