-- Migration: Complete customer level data migration
-- Description: Removes hardcoded CHECK constraint on customers.level, ensures all customers have level_id set

-- Step 1: Drop the hardcoded CHECK constraint on customers.level
-- This allows tenants to have custom level codes beyond the original 5 hardcoded values
ALTER TABLE customers DROP CONSTRAINT IF EXISTS chk_customer_level;

-- Step 2: Ensure all tenants have default customer levels
-- This handles tenants that were created after the initial migration 000026
-- and may not have customer levels defined yet

-- Insert default levels for any tenant that doesn't have any customer levels
INSERT INTO customer_levels (id, tenant_id, code, name, discount_rate, sort_order, is_default, is_active, description)
SELECT
    gen_random_uuid(),
    t.id,
    unnest(ARRAY['normal', 'silver', 'gold', 'platinum', 'vip']),
    unnest(ARRAY['普通会员', '银卡会员', '金卡会员', '白金会员', 'VIP会员']),
    unnest(ARRAY[0.0000, 0.0300, 0.0500, 0.0800, 0.1000]::DECIMAL[]),
    unnest(ARRAY[0, 1, 2, 3, 4]),
    unnest(ARRAY[true, false, false, false, false]),
    true,
    unnest(ARRAY['普通会员，无折扣', '银卡会员，享受3%折扣', '金卡会员，享受5%折扣', '白金会员，享受8%折扣', 'VIP会员，享受10%折扣'])
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM customer_levels cl WHERE cl.tenant_id = t.id
)
ON CONFLICT (tenant_id, code) DO NOTHING;

-- Step 3: Update any customers without level_id to reference their level from customer_levels
-- This ensures data integrity for customers that existed before the level_id column was added
UPDATE customers c
SET level_id = cl.id
FROM customer_levels cl
WHERE c.tenant_id = cl.tenant_id
  AND c.level = cl.code
  AND c.level_id IS NULL;

-- Step 4: For customers whose level code doesn't exist in customer_levels for their tenant,
-- assign them the default level for their tenant
UPDATE customers c
SET level_id = cl.id
FROM customer_levels cl
WHERE c.tenant_id = cl.tenant_id
  AND cl.is_default = TRUE
  AND c.level_id IS NULL;

-- Step 5: As a final fallback, for any remaining customers without level_id,
-- create a 'normal' level for their tenant and assign it
-- First, insert 'normal' level for tenants that have customers without level_id
INSERT INTO customer_levels (id, tenant_id, code, name, discount_rate, sort_order, is_default, is_active, description)
SELECT DISTINCT
    gen_random_uuid(),
    c.tenant_id,
    'normal',
    '普通会员',
    0.0000,
    0,
    true,
    true,
    '普通会员，无折扣'
FROM customers c
WHERE c.level_id IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM customer_levels cl
    WHERE cl.tenant_id = c.tenant_id AND cl.code = 'normal'
  )
ON CONFLICT (tenant_id, code) DO NOTHING;

-- Then assign the 'normal' level to remaining customers
UPDATE customers c
SET level_id = cl.id
FROM customer_levels cl
WHERE c.tenant_id = cl.tenant_id
  AND cl.code = 'normal'
  AND c.level_id IS NULL;

-- Step 6: Sync the denormalized level column from level_id for data consistency
-- This updates the level varchar column to match the code from customer_levels
UPDATE customers c
SET level = cl.code
FROM customer_levels cl
WHERE c.level_id = cl.id
  AND c.level != cl.code;

-- Step 7: Add a comment explaining the migration
COMMENT ON COLUMN customers.level IS 'Customer level code (denormalized from customer_levels for performance). Use level_id for authoritative reference.';

-- Step 8: Create a trigger to keep level column in sync with level_id
CREATE OR REPLACE FUNCTION sync_customer_level_code()
RETURNS TRIGGER AS $$
BEGIN
    -- When level_id changes, update the level code
    IF NEW.level_id IS NOT NULL AND (OLD.level_id IS NULL OR NEW.level_id != OLD.level_id) THEN
        SELECT code INTO NEW.level
        FROM customer_levels
        WHERE id = NEW.level_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sync_customer_level_code_trigger
    BEFORE INSERT OR UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION sync_customer_level_code();

-- Verification query (not executed, but useful for manual verification):
-- SELECT
--     COUNT(*) AS total_customers,
--     COUNT(level_id) AS with_level_id,
--     COUNT(*) - COUNT(level_id) AS without_level_id
-- FROM customers;
