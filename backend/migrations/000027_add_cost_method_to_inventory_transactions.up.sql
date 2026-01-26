-- Add cost_method column to inventory_transactions table
-- Records the cost calculation method used for each transaction (e.g., "moving_average", "fifo")

ALTER TABLE inventory_transactions
ADD COLUMN IF NOT EXISTS cost_method VARCHAR(30);

-- Add comment for documentation
COMMENT ON COLUMN inventory_transactions.cost_method IS 'Cost calculation method used for this transaction (e.g., moving_average, fifo)';
