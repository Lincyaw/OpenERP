-- Seed Data for ERP Test Environment
-- This file contains sample data for all EXISTING modules to enable API smoke testing
-- Run after migrations have been applied
--
-- NOTE: This seed data matches the current migration state.
-- Tables like users, roles, sales_orders, purchase_orders are NOT yet migrated,
-- so their seed data is commented out for future use.

-- ============================================================================
-- TENANT DATA (Multi-tenancy foundation)
-- ============================================================================

-- Note: Default tenant is already created in migration 000001_init_schema.up.sql
-- UUID: 00000000-0000-0000-0000-000000000001

-- Insert additional test tenants
INSERT INTO tenants (id, name, code, status, settings) VALUES
('00000000-0000-0000-0000-000000000002', 'Test Company Alpha', 'alpha', 'active', '{"timezone": "Asia/Shanghai", "currency": "CNY"}'),
('00000000-0000-0000-0000-000000000003', 'Test Company Beta', 'beta', 'active', '{"timezone": "Asia/Shanghai", "currency": "CNY"}')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- IDENTITY MODULE - Test Users and Role Permissions
-- ============================================================================

-- Test users for E2E testing (password: admin123 - bcrypt hash)
-- Note: admin user is created in migration 000017_create_users.up.sql
INSERT INTO users (id, tenant_id, username, password_hash, display_name, status) VALUES
('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'sales', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Sales Manager', 'active'),
('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'warehouse', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Warehouse Manager', 'active'),
('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'finance', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Finance Manager', 'active')
ON CONFLICT (tenant_id, username) DO NOTHING;

-- Assign roles to test users
INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES
-- sales user gets SALES role
('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001'),
-- warehouse user gets WAREHOUSE role
('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001'),
-- finance user gets ACCOUNTANT role
('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;

-- Add permissions to SALES role (sales operations + customer/product read)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000012',
    '00000000-0000-0000-0000-000000000001',
    p.code,
    p.resource,
    p.action,
    p.description
FROM (VALUES
    ('product:read', 'product', 'read', 'View products'),
    ('category:read', 'category', 'read', 'View categories'),
    ('customer:read', 'customer', 'read', 'View customers'),
    ('customer:create', 'customer', 'create', 'Create customers'),
    ('customer:update', 'customer', 'update', 'Update customers'),
    ('sales_order:read', 'sales_order', 'read', 'View sales orders'),
    ('sales_order:create', 'sales_order', 'create', 'Create sales orders'),
    ('sales_order:update', 'sales_order', 'update', 'Update sales orders'),
    ('sales_order:delete', 'sales_order', 'delete', 'Delete sales orders'),
    ('inventory:read', 'inventory', 'read', 'View inventory'),
    ('account_receivable:read', 'account_receivable', 'read', 'View receivables'),
    ('receipt:create', 'receipt', 'create', 'Create receipts'),
    ('receipt:read', 'receipt', 'read', 'View receipts'),
    ('report:read', 'report', 'read', 'View reports')
) AS p(code, resource, action, description)
ON CONFLICT DO NOTHING;

-- Add permissions to WAREHOUSE role (inventory operations + product read)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000014',
    '00000000-0000-0000-0000-000000000001',
    p.code,
    p.resource,
    p.action,
    p.description
FROM (VALUES
    ('product:read', 'product', 'read', 'View products'),
    ('category:read', 'category', 'read', 'View categories'),
    ('warehouse:read', 'warehouse', 'read', 'View warehouses'),
    ('inventory:read', 'inventory', 'read', 'View inventory'),
    ('inventory:adjust', 'inventory', 'adjust', 'Adjust inventory'),
    ('inventory:lock', 'inventory', 'lock', 'Lock inventory'),
    ('inventory:unlock', 'inventory', 'unlock', 'Unlock inventory'),
    ('purchase_order:receive', 'purchase_order', 'receive', 'Receive goods'),
    ('purchase_order:read', 'purchase_order', 'read', 'View purchase orders')
) AS p(code, resource, action, description)
ON CONFLICT DO NOTHING;

-- Add permissions to ACCOUNTANT role (finance operations)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000016',
    '00000000-0000-0000-0000-000000000001',
    p.code,
    p.resource,
    p.action,
    p.description
FROM (VALUES
    ('account_receivable:read', 'account_receivable', 'read', 'View receivables'),
    ('account_receivable:reconcile', 'account_receivable', 'reconcile', 'Reconcile receivables'),
    ('account_payable:read', 'account_payable', 'read', 'View payables'),
    ('account_payable:reconcile', 'account_payable', 'reconcile', 'Reconcile payables'),
    ('receipt:create', 'receipt', 'create', 'Create receipts'),
    ('receipt:read', 'receipt', 'read', 'View receipts'),
    ('payment:create', 'payment', 'create', 'Create payments'),
    ('payment:read', 'payment', 'read', 'View payments'),
    ('expense:create', 'expense', 'create', 'Create expenses'),
    ('expense:read', 'expense', 'read', 'View expenses'),
    ('expense:update', 'expense', 'update', 'Update expenses'),
    ('expense:delete', 'expense', 'delete', 'Delete expenses'),
    ('income:create', 'income', 'create', 'Create income'),
    ('income:read', 'income', 'read', 'View income'),
    ('income:update', 'income', 'update', 'Update income'),
    ('income:delete', 'income', 'delete', 'Delete income'),
    ('report:read', 'report', 'read', 'View reports'),
    ('report:export', 'report', 'export', 'Export reports'),
    ('customer:read', 'customer', 'read', 'View customers'),
    ('supplier:read', 'supplier', 'read', 'View suppliers')
) AS p(code, resource, action, description)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- CATALOG MODULE - Categories and Products
-- ============================================================================

-- Root categories
INSERT INTO categories (id, tenant_id, code, name, description, parent_id, path, level, sort_order, status) VALUES
('30000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'ELEC', 'Electronics', 'Electronic devices and accessories', NULL, '30000000-0000-0000-0000-000000000001', 0, 1, 'active'),
('30000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'CLOTH', 'Clothing', 'Apparel and fashion items', NULL, '30000000-0000-0000-0000-000000000002', 0, 2, 'active'),
('30000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'FOOD', 'Food & Beverages', 'Food products and drinks', NULL, '30000000-0000-0000-0000-000000000003', 0, 3, 'active'),
('30000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'OFFICE', 'Office Supplies', 'Office equipment and supplies', NULL, '30000000-0000-0000-0000-000000000004', 0, 4, 'active')
ON CONFLICT DO NOTHING;

-- Sub-categories
INSERT INTO categories (id, tenant_id, code, name, description, parent_id, path, level, sort_order, status) VALUES
-- Electronics sub-categories
('30000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'PHONE', 'Mobile Phones', 'Smartphones and accessories', '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001/30000000-0000-0000-0000-000000000011', 1, 1, 'active'),
('30000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'COMP', 'Computers', 'Laptops and desktops', '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001/30000000-0000-0000-0000-000000000012', 1, 2, 'active'),
('30000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', 'ACC', 'Accessories', 'Electronic accessories', '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001/30000000-0000-0000-0000-000000000013', 1, 3, 'active'),
-- Clothing sub-categories
('30000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000001', 'MENS', 'Men''s Wear', 'Men''s clothing', '30000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002/30000000-0000-0000-0000-000000000021', 1, 1, 'active'),
('30000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000001', 'WOMENS', 'Women''s Wear', 'Women''s clothing', '30000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002/30000000-0000-0000-0000-000000000022', 1, 2, 'active')
ON CONFLICT DO NOTHING;

-- Products
INSERT INTO products (id, tenant_id, code, name, description, barcode, category_id, unit, purchase_price, selling_price, min_stock, status, sort_order) VALUES
-- Electronics - Phones
('40000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'IPHONE15', 'iPhone 15 Pro', 'Apple iPhone 15 Pro 256GB', '6941234567890', '30000000-0000-0000-0000-000000000011', 'pcs', 7000.0000, 8999.0000, 10, 'active', 1),
('40000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'SAMSUNG24', 'Samsung Galaxy S24', 'Samsung Galaxy S24 Ultra', '6941234567891', '30000000-0000-0000-0000-000000000011', 'pcs', 6000.0000, 7999.0000, 10, 'active', 2),
('40000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'XIAOMI14', 'Xiaomi 14 Pro', 'Xiaomi 14 Pro 512GB', '6941234567892', '30000000-0000-0000-0000-000000000011', 'pcs', 3500.0000, 4999.0000, 15, 'active', 3),
-- Electronics - Computers
('40000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'MACPRO', 'MacBook Pro 14', 'Apple MacBook Pro 14" M3', '6941234567893', '30000000-0000-0000-0000-000000000012', 'pcs', 12000.0000, 14999.0000, 5, 'active', 1),
('40000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'THINKPAD', 'ThinkPad X1 Carbon', 'Lenovo ThinkPad X1 Carbon Gen 11', '6941234567894', '30000000-0000-0000-0000-000000000012', 'pcs', 8000.0000, 10999.0000, 5, 'active', 2),
-- Accessories
('40000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'AIRPODS', 'AirPods Pro', 'Apple AirPods Pro 2nd Gen', '6941234567895', '30000000-0000-0000-0000-000000000013', 'pcs', 1200.0000, 1799.0000, 20, 'active', 1),
('40000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'CHARGER', 'USB-C Charger 65W', 'Universal USB-C 65W Fast Charger', '6941234567896', '30000000-0000-0000-0000-000000000013', 'pcs', 50.0000, 99.0000, 50, 'active', 2),
-- Clothing - Men's
('40000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'TSHIRT-M', 'Men''s Cotton T-Shirt', 'Premium cotton t-shirt', '6941234567897', '30000000-0000-0000-0000-000000000021', 'pcs', 30.0000, 79.0000, 100, 'active', 1),
('40000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'JEANS-M', 'Men''s Denim Jeans', 'Classic fit denim jeans', '6941234567898', '30000000-0000-0000-0000-000000000021', 'pcs', 80.0000, 199.0000, 50, 'active', 2),
-- Office Supplies
('40000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'PAPER-A4', 'A4 Copy Paper', 'A4 copy paper 500 sheets', '6941234567899', '30000000-0000-0000-0000-000000000004', 'pack', 15.0000, 29.0000, 200, 'active', 1)
ON CONFLICT DO NOTHING;

-- Product units (multi-unit support)
INSERT INTO product_units (id, tenant_id, product_id, name, ratio, is_base_unit, is_default_purchase, is_default_sales) VALUES
('41000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'pack', 1.0000, true, false, true),
('41000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'box', 5.0000, false, true, false),
('41000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'carton', 20.0000, false, false, false)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- PARTNER MODULE - Customers, Suppliers, Warehouses
-- ============================================================================

-- Customers
INSERT INTO customers (id, tenant_id, code, name, short_name, type, level, status, contact_name, phone, email, address, city, province, credit_limit, balance) VALUES
('50000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'CUST001', 'Beijing Tech Solutions Ltd', 'Beijing Tech', 'organization', 'gold', 'active', 'Zhang Wei', '13800138001', 'zhang.wei@bjtech.com', 'No. 100 Zhongguancun Street', 'Beijing', 'Beijing', 100000.0000, 5000.0000),
('50000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'CUST002', 'Shanghai Digital Corp', 'Shanghai Digital', 'organization', 'platinum', 'active', 'Li Ming', '13800138002', 'li.ming@shdigital.com', 'No. 200 Nanjing Road', 'Shanghai', 'Shanghai', 200000.0000, 10000.0000),
('50000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'CUST003', 'Shenzhen Hardware Inc', 'SZ Hardware', 'organization', 'silver', 'active', 'Wang Fang', '13800138003', 'wang.fang@szhw.com', 'No. 300 Science Park', 'Shenzhen', 'Guangdong', 50000.0000, 0.0000),
('50000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'CUST004', 'Chen Xiaoming', 'Chen', 'individual', 'normal', 'active', 'Chen Xiaoming', '13800138004', 'chen.xiaoming@gmail.com', 'Room 501, Building A', 'Hangzhou', 'Zhejiang', 10000.0000, 500.0000),
('50000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'CUST005', 'Wang Xiaohong', 'Wang', 'individual', 'vip', 'active', 'Wang Xiaohong', '13800138005', 'wang.xiaohong@qq.com', 'No. 88 Tianhe Road', 'Guangzhou', 'Guangdong', 20000.0000, 2000.0000)
ON CONFLICT DO NOTHING;

-- Suppliers
INSERT INTO suppliers (id, tenant_id, code, name, short_name, type, status, contact_name, phone, email, address, city, province, tax_id, bank_name, bank_account, credit_days, credit_limit, rating) VALUES
('51000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'SUP001', 'Apple China Distribution', 'Apple China', 'distributor', 'active', 'Liu Yang', '13900139001', 'liu.yang@apple-china.com', 'No. 1 Apple Road', 'Shanghai', 'Shanghai', '91310000MA1FL3Q86N', 'ICBC', '6222021234567890123', 30, 1000000.0000, 5),
('51000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'SUP002', 'Samsung Electronics China', 'Samsung China', 'manufacturer', 'active', 'Park Jinhui', '13900139002', 'park.jinhui@samsung.cn', 'No. 88 Suzhou Industrial Park', 'Suzhou', 'Jiangsu', '91320500MA1N6J7X9K', 'BOC', '6222021234567890124', 45, 800000.0000, 4),
('51000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'SUP003', 'Xiaomi Technology Ltd', 'Xiaomi', 'manufacturer', 'active', 'Huang Lei', '13900139003', 'huang.lei@xiaomi.com', 'No. 68 Qinghe Middle Street', 'Beijing', 'Beijing', '91110108MA01DFLF88', 'CMB', '6222021234567890125', 30, 500000.0000, 4),
('51000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'SUP004', 'Lenovo Group China', 'Lenovo', 'manufacturer', 'active', 'Chen Mei', '13900139004', 'chen.mei@lenovo.com.cn', 'No. 6 Chuangye Road', 'Beijing', 'Beijing', '91110000600004685F', 'ABC', '6222021234567890126', 60, 600000.0000, 5),
('51000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'SUP005', 'General Supplies Trading', 'General Supplies', 'distributor', 'active', 'Zhou Qiang', '13900139005', 'zhou.qiang@gsupplies.com', 'No. 200 Warehouse District', 'Guangzhou', 'Guangdong', '91440101MA5CLQJ05P', 'CCB', '6222021234567890127', 15, 100000.0000, 3)
ON CONFLICT DO NOTHING;

-- Warehouses
INSERT INTO warehouses (id, tenant_id, code, name, short_name, type, status, contact_name, phone, email, address, city, province, is_default, capacity) VALUES
('52000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'WH001', 'Main Warehouse Beijing', 'Beijing Main', 'physical', 'active', 'Sun Liang', '13700137001', 'sun.liang@warehouse.com', 'No. 1 Logistics Park', 'Beijing', 'Beijing', true, 10000),
('52000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'WH002', 'Shanghai Distribution Center', 'Shanghai DC', 'physical', 'active', 'Zhao Jie', '13700137002', 'zhao.jie@warehouse.com', 'No. 88 Pudong Logistics', 'Shanghai', 'Shanghai', false, 8000),
('52000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'WH003', 'Shenzhen Warehouse', 'Shenzhen WH', 'physical', 'active', 'Wu Ting', '13700137003', 'wu.ting@warehouse.com', 'No. 50 Baoan Industrial', 'Shenzhen', 'Guangdong', false, 5000),
('52000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'WH-VIRTUAL', 'Virtual Inventory', 'Virtual', 'virtual', 'active', NULL, NULL, NULL, NULL, NULL, NULL, false, 0)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- INVENTORY MODULE - Inventory Items and Stock Batches
-- ============================================================================

-- Inventory items (linking products to warehouses)
INSERT INTO inventory_items (id, tenant_id, warehouse_id, product_id, available_quantity, locked_quantity, total_quantity, unit_cost, min_stock_threshold, max_stock_threshold, status) VALUES
-- Main Warehouse Beijing inventory
('60000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 50.0000, 5.0000, 55.0000, 7000.0000, 10.0000, 100.0000, 'active'),
('60000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000002', 30.0000, 0.0000, 30.0000, 6000.0000, 10.0000, 80.0000, 'active'),
('60000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000003', 100.0000, 10.0000, 110.0000, 3500.0000, 15.0000, 150.0000, 'active'),
('60000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000004', 20.0000, 2.0000, 22.0000, 12000.0000, 5.0000, 50.0000, 'active'),
('60000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000006', 80.0000, 5.0000, 85.0000, 1200.0000, 20.0000, 200.0000, 'active'),
-- Shanghai DC inventory
('60000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 40.0000, 0.0000, 40.0000, 7000.0000, 10.0000, 100.0000, 'active'),
('60000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000002', 25.0000, 3.0000, 28.0000, 6000.0000, 10.0000, 80.0000, 'active'),
('60000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000005', 15.0000, 0.0000, 15.0000, 8000.0000, 5.0000, 40.0000, 'active'),
-- Shenzhen warehouse inventory
('60000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000003', 200.0000, 20.0000, 220.0000, 3500.0000, 15.0000, 300.0000, 'active'),
('60000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000007', 500.0000, 0.0000, 500.0000, 50.0000, 50.0000, 1000.0000, 'active')
ON CONFLICT DO NOTHING;

-- Stock batches
INSERT INTO stock_batches (id, tenant_id, inventory_item_id, batch_number, initial_quantity, current_quantity, unit_cost, received_at, expires_at, status) VALUES
-- iPhone 15 batches in Beijing
('61000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 'BATCH-IP15-001', 30.0000, 25.0000, 7000.0000, NOW() - INTERVAL '30 days', NULL, 'active'),
('61000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 'BATCH-IP15-002', 30.0000, 30.0000, 7000.0000, NOW() - INTERVAL '15 days', NULL, 'active'),
-- Xiaomi 14 batches in Beijing
('61000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', 'BATCH-XM14-001', 60.0000, 50.0000, 3500.0000, NOW() - INTERVAL '45 days', NULL, 'active'),
('61000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', 'BATCH-XM14-002', 60.0000, 60.0000, 3500.0000, NOW() - INTERVAL '20 days', NULL, 'active')
ON CONFLICT DO NOTHING;

-- Stock locks (for pending orders - reference_id can be placeholder since trade tables don't exist yet)
INSERT INTO stock_locks (id, tenant_id, inventory_item_id, reference_type, reference_id, quantity, locked_at, expires_at, status) VALUES
('62000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 'sales_order', '70000000-0000-0000-0000-000000000001', 5.0000, NOW(), NOW() + INTERVAL '7 days', 'active'),
('62000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', 'sales_order', '70000000-0000-0000-0000-000000000002', 10.0000, NOW(), NOW() + INTERVAL '7 days', 'active')
ON CONFLICT DO NOTHING;

-- Inventory transactions
INSERT INTO inventory_transactions (id, tenant_id, inventory_item_id, transaction_type, quantity, unit_cost, reference_type, reference_id, batch_number, notes) VALUES
-- Initial stock entries
('90000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 'in', 55.0000, 7000.0000, 'manual', NULL, 'BATCH-IP15-001', 'Initial stock entry'),
('90000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', 'in', 110.0000, 3500.0000, 'manual', NULL, 'BATCH-XM14-001', 'Initial stock entry'),
('90000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000002', 'in', 30.0000, 6000.0000, 'manual', NULL, NULL, 'Initial Samsung stock'),
('90000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000004', 'in', 22.0000, 12000.0000, 'manual', NULL, NULL, 'Initial MacBook stock')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- FINANCE MODULE - Receivables, Payables, Vouchers
-- ============================================================================

-- Account receivables (without foreign key to sales_orders since that table doesn't exist)
INSERT INTO account_receivables (id, tenant_id, receivable_number, customer_id, source_type, source_id, original_amount, paid_amount, balance, status, due_date) VALUES
('80000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'AR-2026-0001', '50000000-0000-0000-0000-000000000001', 'manual', NULL, 53994.0000, 0.0000, 53994.0000, 'pending', NOW() + INTERVAL '30 days'),
('80000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'AR-2026-0002', '50000000-0000-0000-0000-000000000002', 'manual', NULL, 49000.0000, 10000.0000, 39000.0000, 'partial', NOW() + INTERVAL '45 days'),
('80000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'AR-2026-0003', '50000000-0000-0000-0000-000000000004', 'manual', NULL, 8999.0000, 8999.0000, 0.0000, 'paid', NOW() - INTERVAL '5 days'),
('80000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'AR-2026-0004', '50000000-0000-0000-0000-000000000005', 'manual', NULL, 14999.0000, 14999.0000, 0.0000, 'paid', NOW() - INTERVAL '10 days')
ON CONFLICT DO NOTHING;

-- Account payables (without foreign key to purchase_orders since that table doesn't exist)
INSERT INTO account_payables (id, tenant_id, payable_number, supplier_id, source_type, source_id, original_amount, paid_amount, balance, status, due_date) VALUES
('81000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'AP-2026-0001', '51000000-0000-0000-0000-000000000001', 'manual', NULL, 210000.0000, 210000.0000, 0.0000, 'paid', NOW() - INTERVAL '5 days'),
('81000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'AP-2026-0002', '51000000-0000-0000-0000-000000000003', 'manual', NULL, 210000.0000, 50000.0000, 160000.0000, 'partial', NOW() + INTERVAL '5 days'),
('81000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'AP-2026-0003', '51000000-0000-0000-0000-000000000002', 'manual', NULL, 150000.0000, 0.0000, 150000.0000, 'pending', NOW() + INTERVAL '40 days')
ON CONFLICT DO NOTHING;

-- Receipt vouchers (customer payments received)
INSERT INTO receipt_vouchers (id, tenant_id, voucher_number, customer_id, amount, payment_method, status, receipt_date, notes) VALUES
('82000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'RV-2026-0001', '50000000-0000-0000-0000-000000000002', 10000.0000, 'bank_transfer', 'confirmed', NOW() - INTERVAL '2 days', 'Partial payment'),
('82000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'RV-2026-0002', '50000000-0000-0000-0000-000000000004', 8999.0000, 'wechat', 'confirmed', NOW() - INTERVAL '8 days', 'Full payment'),
('82000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'RV-2026-0003', '50000000-0000-0000-0000-000000000005', 14999.0000, 'alipay', 'confirmed', NOW() - INTERVAL '12 days', 'Full payment')
ON CONFLICT DO NOTHING;

-- Payment vouchers (payments to suppliers)
INSERT INTO payment_vouchers (id, tenant_id, voucher_number, supplier_id, amount, payment_method, status, payment_date, notes) VALUES
('83000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'PV-2026-0001', '51000000-0000-0000-0000-000000000001', 210000.0000, 'bank_transfer', 'confirmed', NOW() - INTERVAL '8 days', 'Full payment'),
('83000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'PV-2026-0002', '51000000-0000-0000-0000-000000000003', 50000.0000, 'bank_transfer', 'confirmed', NOW() - INTERVAL '3 days', 'Partial payment')
ON CONFLICT DO NOTHING;

-- Expense records
INSERT INTO expense_records (id, tenant_id, record_number, category, amount, expense_date, description, status) VALUES
('84000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0001', 'rent', 15000.0000, NOW() - INTERVAL '15 days', 'Monthly warehouse rent - Beijing', 'confirmed'),
('84000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0002', 'utilities', 3500.0000, NOW() - INTERVAL '10 days', 'Electricity and water bills', 'confirmed'),
('84000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0003', 'logistics', 8000.0000, NOW() - INTERVAL '5 days', 'Shipping costs for January', 'confirmed'),
('84000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0004', 'salary', 50000.0000, NOW() - INTERVAL '1 day', 'Staff salaries', 'draft')
ON CONFLICT DO NOTHING;

-- Other income records
INSERT INTO other_income_records (id, tenant_id, record_number, category, amount, income_date, description, status) VALUES
('85000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'INC-2026-0001', 'service', 5000.0000, NOW() - INTERVAL '12 days', 'Installation service fee', 'confirmed'),
('85000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'INC-2026-0002', 'interest', 1200.0000, NOW() - INTERVAL '8 days', 'Bank interest income', 'confirmed')
ON CONFLICT DO NOTHING;

-- Balance transactions (customer prepaid balance changes)
INSERT INTO balance_transactions (id, tenant_id, customer_id, type, amount, balance_before, balance_after, reference_type, reference_id, notes) VALUES
('86000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000001', 'recharge', 5000.0000, 0.0000, 5000.0000, 'manual', NULL, 'Initial deposit'),
('86000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000002', 'recharge', 10000.0000, 0.0000, 10000.0000, 'manual', NULL, 'VIP customer deposit'),
('86000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000004', 'recharge', 500.0000, 0.0000, 500.0000, 'manual', NULL, 'Small recharge'),
('86000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000005', 'recharge', 3000.0000, 0.0000, 3000.0000, 'manual', NULL, 'VIP deposit'),
('86000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000005', 'consumption', -1000.0000, 3000.0000, 2000.0000, 'manual', NULL, 'Used for order payment')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SUMMARY
-- ============================================================================
-- Test data created for existing tables:
-- - 3 tenants (default + 2 test companies)
-- - 4 root categories + 5 sub-categories (9 total)
-- - 10 products with various prices
-- - 3 product units for A4 paper (multi-unit demo)
-- - 5 customers (3 organizations, 2 individuals)
-- - 5 suppliers
-- - 4 warehouses (3 physical, 1 virtual)
-- - 10 inventory items across warehouses
-- - 4 stock batches
-- - 2 stock locks
-- - 4 inventory transactions
-- - 4 account receivables
-- - 3 account payables
-- - 3 receipt vouchers
-- - 2 payment vouchers
-- - 4 expense records
-- - 2 other income records
-- - 5 balance transactions
--
-- NOTE: Tables for users, roles, sales_orders, purchase_orders, sales_returns,
-- purchase_returns are NOT yet migrated. When those migrations are added,
-- uncomment the relevant seed data sections above.
