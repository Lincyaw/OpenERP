-- Remove unit conversion fields from sales_return_items table
-- This migration rolls back the addition of ConversionRate, BaseQuantity, and BaseUnit columns

ALTER TABLE sales_return_items
DROP COLUMN IF EXISTS conversion_rate;

ALTER TABLE sales_return_items
DROP COLUMN IF EXISTS base_quantity;

ALTER TABLE sales_return_items
DROP COLUMN IF EXISTS base_unit;
