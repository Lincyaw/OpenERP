-- Migration: Revert customer level data migration
-- Description: Restores the hardcoded CHECK constraint and removes the sync trigger

-- Step 1: Drop the sync trigger
DROP TRIGGER IF EXISTS sync_customer_level_code_trigger ON customers;
DROP FUNCTION IF EXISTS sync_customer_level_code();

-- Step 2: Restore the CHECK constraint on customers.level
-- Note: This will fail if any customers have level codes not in the original set
-- First, update any non-standard levels to 'normal' to avoid constraint violation
UPDATE customers
SET level = 'normal'
WHERE level NOT IN ('normal', 'silver', 'gold', 'platinum', 'vip');

-- Then add back the CHECK constraint
ALTER TABLE customers ADD CONSTRAINT chk_customer_level
    CHECK (level IN ('normal', 'silver', 'gold', 'platinum', 'vip'));

-- Step 3: Restore the original comment
COMMENT ON COLUMN customers.level IS 'Customer tier: normal, silver, gold, platinum, vip';
