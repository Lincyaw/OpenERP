-- Drop purchase_return_items first (has foreign key to purchase_returns)
DROP TABLE IF EXISTS purchase_return_items;

-- Drop purchase_returns (has foreign key to purchase_orders)
DROP TABLE IF EXISTS purchase_returns;

-- Drop sales_return_items first (has foreign key to sales_returns)
DROP TABLE IF EXISTS sales_return_items;

-- Drop sales_returns (has foreign key to sales_orders)
DROP TABLE IF EXISTS sales_returns;

-- Drop purchase_order_items first (has foreign key to purchase_orders)
DROP TABLE IF EXISTS purchase_order_items;

-- Drop purchase_orders
DROP TABLE IF EXISTS purchase_orders;

-- Drop sales_order_items first (has foreign key to sales_orders)
DROP TABLE IF EXISTS sales_order_items;

-- Drop sales_orders
DROP TABLE IF EXISTS sales_orders;
