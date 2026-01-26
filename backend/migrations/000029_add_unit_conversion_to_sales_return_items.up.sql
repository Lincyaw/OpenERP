-- Add unit conversion fields to sales_return_items table
-- This migration adds ConversionRate, BaseQuantity, and BaseUnit columns
-- to support proper inventory restoration using base units

-- Add conversion_rate column with default value of 1
ALTER TABLE sales_return_items
ADD COLUMN IF NOT EXISTS conversion_rate DECIMAL(18,6) NOT NULL DEFAULT 1;

-- Add base_quantity column (will be calculated from return_quantity * conversion_rate)
ALTER TABLE sales_return_items
ADD COLUMN IF NOT EXISTS base_quantity DECIMAL(18,4) NOT NULL DEFAULT 0;

-- Add base_unit column
ALTER TABLE sales_return_items
ADD COLUMN IF NOT EXISTS base_unit VARCHAR(20) NOT NULL DEFAULT '';

-- Update existing records: set base_quantity = return_quantity (assuming conversion_rate = 1)
-- and base_unit = unit (assuming they were using base units)
UPDATE sales_return_items
SET base_quantity = return_quantity,
    base_unit = unit
WHERE base_quantity = 0 OR base_unit = '';

-- Add comment for documentation
COMMENT ON COLUMN sales_return_items.conversion_rate IS 'Conversion rate from order unit to base unit (1 if using base unit)';
COMMENT ON COLUMN sales_return_items.base_quantity IS 'Return quantity in base units for inventory operations';
COMMENT ON COLUMN sales_return_items.base_unit IS 'Base unit code for inventory operations';
