-- Seed Data for ERP Test Environment
-- Comprehensive data for all modules with proper relationships
-- Run after migrations have been applied

-- ============================================================================
-- LAYER 1: TENANT DATA (Multi-tenancy foundation)
-- ============================================================================

-- Note: Default tenant is already created in migration 000001_init_schema.up.sql
-- UUID: 00000000-0000-0000-0000-000000000001

INSERT INTO tenants (id, name, code, status, settings) VALUES
('00000000-0000-0000-0000-000000000002', 'Test Company Alpha', 'alpha', 'active', '{"timezone": "Asia/Shanghai", "currency": "CNY"}'),
('00000000-0000-0000-0000-000000000003', 'Test Company Beta', 'beta', 'active', '{"timezone": "Asia/Shanghai", "currency": "CNY"}')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- LAYER 1: CATEGORIES (10 total: 4 root + 6 sub)
-- ============================================================================

-- Root categories (4)
INSERT INTO categories (id, tenant_id, code, name, description, parent_id, path, level, sort_order, status) VALUES
('30000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'ELEC', 'Electronics', 'Electronic devices and accessories', NULL, '30000000-0000-0000-0000-000000000001', 0, 1, 'active'),
('30000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'CLOTH', 'Clothing', 'Apparel and fashion items', NULL, '30000000-0000-0000-0000-000000000002', 0, 2, 'active'),
('30000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'FOOD', 'Food & Beverages', 'Food products and drinks', NULL, '30000000-0000-0000-0000-000000000003', 0, 3, 'active'),
('30000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'OFFICE', 'Office Supplies', 'Office equipment and supplies', NULL, '30000000-0000-0000-0000-000000000004', 0, 4, 'active')
ON CONFLICT DO NOTHING;

-- Sub-categories (6)
INSERT INTO categories (id, tenant_id, code, name, description, parent_id, path, level, sort_order, status) VALUES
('30000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'PHONE', 'Mobile Phones', 'Smartphones and accessories', '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001/30000000-0000-0000-0000-000000000011', 1, 1, 'active'),
('30000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'COMP', 'Computers', 'Laptops and desktops', '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001/30000000-0000-0000-0000-000000000012', 1, 2, 'active'),
('30000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', 'ACC', 'Accessories', 'Electronic accessories', '30000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001/30000000-0000-0000-0000-000000000013', 1, 3, 'active'),
('30000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000001', 'MENS', 'Mens Wear', 'Mens clothing', '30000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002/30000000-0000-0000-0000-000000000021', 1, 1, 'active'),
('30000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000001', 'WOMENS', 'Womens Wear', 'Womens clothing', '30000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002/30000000-0000-0000-0000-000000000022', 1, 2, 'active'),
('30000000-0000-0000-0000-000000000031', '00000000-0000-0000-0000-000000000001', 'SNACKS', 'Snacks', 'Snack foods and treats', '30000000-0000-0000-0000-000000000003', '30000000-0000-0000-0000-000000000003/30000000-0000-0000-0000-000000000031', 1, 1, 'active')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 2: CUSTOMER LEVELS (15 total: 5 per tenant)
-- ============================================================================

INSERT INTO customer_levels (id, tenant_id, code, name, discount_rate, sort_order, is_default, is_active, description) VALUES
-- Default tenant levels
('60000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'normal', 'Normal Member', 0.0000, 0, true, true, 'Normal member, no discount'),
('60000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'silver', 'Silver Member', 0.0300, 1, false, true, 'Silver member, 3% discount'),
('60000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'gold', 'Gold Member', 0.0500, 2, false, true, 'Gold member, 5% discount'),
('60000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'platinum', 'Platinum Member', 0.0800, 3, false, true, 'Platinum member, 8% discount'),
('60000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'vip', 'VIP Member', 0.1000, 4, false, true, 'VIP member, 10% discount'),
-- Alpha tenant levels
('60000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002', 'normal', 'Normal Member', 0.0000, 0, true, true, 'Normal member'),
('60000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000002', 'silver', 'Silver Member', 0.0300, 1, false, true, 'Silver member'),
('60000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000002', 'gold', 'Gold Member', 0.0500, 2, false, true, 'Gold member'),
('60000000-0000-0000-0000-000000000104', '00000000-0000-0000-0000-000000000002', 'platinum', 'Platinum Member', 0.0800, 3, false, true, 'Platinum member'),
('60000000-0000-0000-0000-000000000105', '00000000-0000-0000-0000-000000000002', 'vip', 'VIP Member', 0.1000, 4, false, true, 'VIP member'),
-- Beta tenant levels
('60000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000003', 'normal', 'Normal Member', 0.0000, 0, true, true, 'Normal member'),
('60000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000003', 'silver', 'Silver Member', 0.0300, 1, false, true, 'Silver member'),
('60000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000003', 'gold', 'Gold Member', 0.0500, 2, false, true, 'Gold member'),
('60000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000003', 'platinum', 'Platinum Member', 0.0800, 3, false, true, 'Platinum member'),
('60000000-0000-0000-0000-000000000205', '00000000-0000-0000-0000-000000000003', 'vip', 'VIP Member', 0.1000, 4, false, true, 'VIP member')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 2: PRODUCTS (15 total)
-- ============================================================================

INSERT INTO products (id, tenant_id, code, name, description, barcode, category_id, unit, purchase_price, selling_price, min_stock, status, sort_order) VALUES
-- Electronics - Phones (3)
('40000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'IPHONE15', 'iPhone 15 Pro', 'Apple iPhone 15 Pro 256GB', '6941234567890', '30000000-0000-0000-0000-000000000011', 'pcs', 7000.0000, 8999.0000, 10, 'active', 1),
('40000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'SAMSUNG24', 'Samsung Galaxy S24', 'Samsung Galaxy S24 Ultra', '6941234567891', '30000000-0000-0000-0000-000000000011', 'pcs', 6000.0000, 7999.0000, 10, 'active', 2),
('40000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'XIAOMI14', 'Xiaomi 14 Pro', 'Xiaomi 14 Pro 512GB', '6941234567892', '30000000-0000-0000-0000-000000000011', 'pcs', 3500.0000, 4999.0000, 15, 'active', 3),
-- Electronics - Computers (3)
('40000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'MACPRO', 'MacBook Pro 14', 'Apple MacBook Pro 14 M3', '6941234567893', '30000000-0000-0000-0000-000000000012', 'pcs', 12000.0000, 14999.0000, 5, 'active', 1),
('40000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'THINKPAD', 'ThinkPad X1 Carbon', 'Lenovo ThinkPad X1 Carbon Gen 11', '6941234567894', '30000000-0000-0000-0000-000000000012', 'pcs', 8000.0000, 10999.0000, 5, 'active', 2),
('40000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'DELLXPS', 'Dell XPS 15', 'Dell XPS 15 9530', '6941234567900', '30000000-0000-0000-0000-000000000012', 'pcs', 9000.0000, 11999.0000, 5, 'active', 3),
-- Electronics - Accessories (3)
('40000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'AIRPODS', 'AirPods Pro', 'Apple AirPods Pro 2nd Gen', '6941234567895', '30000000-0000-0000-0000-000000000013', 'pcs', 1200.0000, 1799.0000, 20, 'active', 1),
('40000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'CHARGER', 'USB-C Charger 65W', 'Universal USB-C 65W Fast Charger', '6941234567896', '30000000-0000-0000-0000-000000000013', 'pcs', 50.0000, 99.0000, 50, 'active', 2),
('40000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'USBHUB', 'USB-C Hub 7-in-1', 'Multi-port USB-C Hub', '6941234567901', '30000000-0000-0000-0000-000000000013', 'pcs', 80.0000, 149.0000, 30, 'active', 3),
-- Clothing - Mens (2)
('40000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'TSHIRT-M', 'Mens Cotton T-Shirt', 'Premium cotton t-shirt', '6941234567897', '30000000-0000-0000-0000-000000000021', 'pcs', 30.0000, 79.0000, 100, 'active', 1),
('40000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'JEANS-M', 'Mens Denim Jeans', 'Classic fit denim jeans', '6941234567898', '30000000-0000-0000-0000-000000000021', 'pcs', 80.0000, 199.0000, 50, 'active', 2),
-- Clothing - Womens (1)
('40000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', 'DRESS-W', 'Womens Summer Dress', 'Elegant summer dress', '6941234567902', '30000000-0000-0000-0000-000000000022', 'pcs', 100.0000, 299.0000, 30, 'active', 1),
-- Office Supplies (2)
('40000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'PAPER-A4', 'A4 Copy Paper', 'A4 copy paper 500 sheets', '6941234567899', '30000000-0000-0000-0000-000000000004', 'pack', 15.0000, 29.0000, 200, 'active', 1),
('40000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', 'PEN-BOX', 'Ballpoint Pen Box', 'Box of 12 ballpoint pens', '6941234567903', '30000000-0000-0000-0000-000000000004', 'box', 10.0000, 25.0000, 100, 'active', 2),
-- Food - Snacks (1)
('40000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000001', 'CHIPS', 'Potato Chips Pack', 'Classic salted potato chips', '6941234567904', '30000000-0000-0000-0000-000000000031', 'pack', 5.0000, 12.0000, 500, 'active', 1)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 2: CUSTOMERS (10 total: 5 organizations + 5 individuals)
-- ============================================================================

INSERT INTO customers (id, tenant_id, code, name, short_name, type, level, level_id, status, contact_name, phone, email, address, city, province, credit_limit, balance) VALUES
-- Organizations (5)
('50000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'CUST001', 'Beijing Tech Solutions Ltd', 'Beijing Tech', 'organization', 'gold', '60000000-0000-0000-0000-000000000003', 'active', 'Zhang Wei', '13800138001', 'zhang.wei@bjtech.com', 'No. 100 Zhongguancun Street', 'Beijing', 'Beijing', 100000.0000, 5000.0000),
('50000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'CUST002', 'Shanghai Digital Corp', 'Shanghai Digital', 'organization', 'platinum', '60000000-0000-0000-0000-000000000004', 'active', 'Li Ming', '13800138002', 'li.ming@shdigital.com', 'No. 200 Nanjing Road', 'Shanghai', 'Shanghai', 200000.0000, 10000.0000),
('50000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'CUST003', 'Shenzhen Hardware Inc', 'SZ Hardware', 'organization', 'silver', '60000000-0000-0000-0000-000000000002', 'active', 'Wang Fang', '13800138003', 'wang.fang@szhw.com', 'No. 300 Science Park', 'Shenzhen', 'Guangdong', 50000.0000, 0.0000),
('50000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'CUST006', 'Guangzhou Trading Co', 'GZ Trading', 'organization', 'vip', '60000000-0000-0000-0000-000000000005', 'active', 'Liu Yang', '13800138006', 'liu.yang@gztrading.com', 'No. 88 Tianhe Road', 'Guangzhou', 'Guangdong', 300000.0000, 15000.0000),
('50000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'CUST007', 'Hangzhou Software Ltd', 'HZ Software', 'organization', 'normal', '60000000-0000-0000-0000-000000000001', 'active', 'Chen Hong', '13800138007', 'chen.hong@hzsoftware.com', 'No. 50 West Lake Road', 'Hangzhou', 'Zhejiang', 80000.0000, 0.0000),
-- Individuals (5)
('50000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'CUST004', 'Chen Xiaoming', 'Chen', 'individual', 'normal', '60000000-0000-0000-0000-000000000001', 'active', 'Chen Xiaoming', '13800138004', 'chen.xiaoming@gmail.com', 'Room 501, Building A', 'Hangzhou', 'Zhejiang', 10000.0000, 500.0000),
('50000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'CUST005', 'Wang Xiaohong', 'Wang', 'individual', 'vip', '60000000-0000-0000-0000-000000000005', 'active', 'Wang Xiaohong', '13800138005', 'wang.xiaohong@qq.com', 'No. 88 Tianhe Road', 'Guangzhou', 'Guangdong', 20000.0000, 2000.0000),
('50000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'CUST008', 'Li Xiaofang', 'Li', 'individual', 'silver', '60000000-0000-0000-0000-000000000002', 'active', 'Li Xiaofang', '13800138008', 'li.xiaofang@163.com', 'Room 202, Tower B', 'Chengdu', 'Sichuan', 15000.0000, 800.0000),
('50000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'CUST009', 'Zhang Xiaoli', 'Zhang', 'individual', 'gold', '60000000-0000-0000-0000-000000000003', 'active', 'Zhang Xiaoli', '13800138009', 'zhang.xiaoli@qq.com', 'No. 168 Jiefang Road', 'Wuhan', 'Hubei', 25000.0000, 1500.0000),
('50000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'CUST010', 'Zhao Xiaojun', 'Zhao', 'individual', 'platinum', '60000000-0000-0000-0000-000000000004', 'active', 'Zhao Xiaojun', '13800138010', 'zhao.xiaojun@hotmail.com', 'No. 99 Binjiang Road', 'Nanjing', 'Jiangsu', 30000.0000, 3000.0000)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 2: SUPPLIERS (8 total)
-- ============================================================================

INSERT INTO suppliers (id, tenant_id, code, name, short_name, type, status, contact_name, phone, email, address, city, province, tax_id, bank_name, bank_account, credit_days, credit_limit, rating) VALUES
('51000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'SUP001', 'Apple China Distribution', 'Apple China', 'distributor', 'active', 'Liu Yang', '13900139001', 'liu.yang@apple-china.com', 'No. 1 Apple Road', 'Shanghai', 'Shanghai', '91310000MA1FL3Q86N', 'ICBC', '6222021234567890123', 30, 1000000.0000, 5),
('51000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'SUP002', 'Samsung Electronics China', 'Samsung China', 'manufacturer', 'active', 'Park Jinhui', '13900139002', 'park.jinhui@samsung.cn', 'No. 88 Suzhou Industrial Park', 'Suzhou', 'Jiangsu', '91320500MA1N6J7X9K', 'BOC', '6222021234567890124', 45, 800000.0000, 4),
('51000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'SUP003', 'Xiaomi Technology Ltd', 'Xiaomi', 'manufacturer', 'active', 'Huang Lei', '13900139003', 'huang.lei@xiaomi.com', 'No. 68 Qinghe Middle Street', 'Beijing', 'Beijing', '91110108MA01DFLF88', 'CMB', '6222021234567890125', 30, 500000.0000, 4),
('51000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'SUP004', 'Lenovo Group China', 'Lenovo', 'manufacturer', 'active', 'Chen Mei', '13900139004', 'chen.mei@lenovo.com.cn', 'No. 6 Chuangye Road', 'Beijing', 'Beijing', '91110000600004685F', 'ABC', '6222021234567890126', 60, 600000.0000, 5),
('51000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'SUP005', 'General Supplies Trading', 'General Supplies', 'distributor', 'active', 'Zhou Qiang', '13900139005', 'zhou.qiang@gsupplies.com', 'No. 200 Warehouse District', 'Guangzhou', 'Guangdong', '91440101MA5CLQJ05P', 'CCB', '6222021234567890127', 15, 100000.0000, 3),
('51000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'SUP006', 'Dell Technologies China', 'Dell China', 'manufacturer', 'active', 'Wang Jing', '13900139006', 'wang.jing@dell.com', 'No. 18 Dell Park', 'Xiamen', 'Fujian', '91350200MA2Y1ABC12', 'PSBC', '6222021234567890128', 45, 700000.0000, 5),
('51000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'SUP007', 'Fashion Textile Co', 'Fashion Textile', 'distributor', 'active', 'Li Hua', '13900139007', 'li.hua@fashiontextile.com', 'No. 99 Textile Avenue', 'Shaoxing', 'Zhejiang', '91330600MA2H4DEF34', 'SPDB', '6222021234567890129', 30, 200000.0000, 4),
('51000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'SUP008', 'Snack Foods International', 'Snack Foods', 'retailer', 'active', 'Zhang Min', '13900139008', 'zhang.min@snackfoods.com', 'No. 66 Food Park', 'Dongguan', 'Guangdong', '91441900MA5KLGHI56', 'CITIC', '6222021234567890130', 15, 150000.0000, 3)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 2: WAREHOUSES (5 total: 3 physical + 1 virtual + 1 returns)
-- ============================================================================

INSERT INTO warehouses (id, tenant_id, code, name, short_name, type, status, contact_name, phone, email, address, city, province, is_default, capacity) VALUES
('52000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'WH001', 'Beijing Main Warehouse', 'Beijing Main', 'physical', 'active', 'Sun Liang', '13700137001', 'sun.liang@warehouse.com', 'Logistics Park No.1', 'Beijing', 'Beijing', true, 10000),
('52000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'WH002', 'Shanghai Distribution Center', 'Shanghai DC', 'physical', 'active', 'Zhao Jie', '13700137002', 'zhao.jie@warehouse.com', 'Pudong Logistics No.88', 'Shanghai', 'Shanghai', false, 8000),
('52000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'WH003', 'Shenzhen Warehouse', 'Shenzhen WH', 'physical', 'active', 'Wu Ting', '13700137003', 'wu.ting@warehouse.com', 'Baoan Industrial Zone No.50', 'Shenzhen', 'Guangdong', false, 5000),
('52000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'WH-VIRTUAL', 'Virtual Inventory', 'Virtual', 'virtual', 'active', NULL, NULL, NULL, NULL, NULL, NULL, false, 0),
('52000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'WH-RETURNS', 'Returns Warehouse', 'Returns WH', 'physical', 'active', 'Ma Qiang', '13700137004', 'ma.qiang@warehouse.com', 'Returns Processing Center', 'Beijing', 'Beijing', false, 2000)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 3: USERS AND ROLES
-- ============================================================================

-- Test users (password: admin123)
INSERT INTO users (id, tenant_id, username, password_hash, display_name, status) VALUES
('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'sales', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Sales Manager', 'active'),
('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'warehouse', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Warehouse Manager', 'active'),
('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'finance', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Finance Manager', 'active'),
('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'purchaser', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Purchaser', 'active'),
('00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'cashier', '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', 'Cashier', 'active')
ON CONFLICT (tenant_id, username) DO NOTHING;

-- Assign roles to test users
INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES
('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001'),
('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001'),
('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000001'),
('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001'),
('00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;

-- Add feature_flag permissions to ADMIN role (for E2E testing)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT '00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', p.code, p.resource, p.action, p.description
FROM (VALUES
    ('feature_flag:read', 'feature_flag', 'read', 'View feature flags'),
    ('feature_flag:create', 'feature_flag', 'create', 'Create feature flags'),
    ('feature_flag:update', 'feature_flag', 'update', 'Update feature flags'),
    ('feature_flag:delete', 'feature_flag', 'delete', 'Delete feature flags'),
    ('feature_flag:evaluate', 'feature_flag', 'evaluate', 'Evaluate feature flags'),
    ('feature_flag:override', 'feature_flag', 'override', 'Manage feature flag overrides'),
    ('feature_flag:audit', 'feature_flag', 'audit', 'View feature flag audit logs'),
    ('feature_flag:admin', 'feature_flag', 'admin', 'Admin access to feature flags')
) AS p(code, resource, action, description)
ON CONFLICT DO NOTHING;

-- Add permissions to SALES role
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT '00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', p.code, p.resource, p.action, p.description
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

-- Add permissions to WAREHOUSE role
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT '00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', p.code, p.resource, p.action, p.description
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

-- Add permissions to ACCOUNTANT role
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT '00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000001', p.code, p.resource, p.action, p.description
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
-- LAYER 4: PRODUCT UNITS (10 total for multi-unit support)
-- ============================================================================

INSERT INTO product_units (id, tenant_id, product_id, unit_code, unit_name, conversion_rate, is_default_purchase_unit, is_default_sales_unit, sort_order) VALUES
-- A4 Paper units (pack/box/carton)
('41000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'pack', 'Pack', 1.0000, false, true, 1),
('41000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'box', 'Box (5 packs)', 5.0000, true, false, 2),
('41000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'carton', 'Carton (20 packs)', 20.0000, false, false, 3),
-- Pen Box units (box/case)
('41000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000014', 'box', 'Box', 1.0000, false, true, 1),
('41000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000014', 'case', 'Case (10 boxes)', 10.0000, true, false, 2),
-- T-Shirt units (pcs/dozen)
('41000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000008', 'pcs', 'Piece', 1.0000, false, true, 1),
('41000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000008', 'dozen', 'Dozen (12 pcs)', 12.0000, true, false, 2),
-- Charger units (pcs/box)
('41000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000007', 'pcs', 'Piece', 1.0000, false, true, 1),
('41000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000007', 'box', 'Box (20 pcs)', 20.0000, true, false, 2),
-- Chips units (pack/carton)
('41000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000015', 'pack', 'Pack', 1.0000, false, true, 1)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 5: INVENTORY ITEMS (20 total across warehouses)
-- ============================================================================

INSERT INTO inventory_items (id, tenant_id, warehouse_id, product_id, available_quantity, locked_quantity, unit_cost, min_quantity, max_quantity) VALUES
-- Beijing Main Warehouse (8 items)
('60000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 50.0000, 5.0000, 7000.0000, 10.0000, 100.0000),
('60000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000002', 30.0000, 3.0000, 6000.0000, 10.0000, 80.0000),
('60000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000003', 100.0000, 10.0000, 3500.0000, 15.0000, 150.0000),
('60000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000004', 20.0000, 2.0000, 12000.0000, 5.0000, 50.0000),
('60000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000006', 80.0000, 5.0000, 1200.0000, 20.0000, 200.0000),
('60000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000007', 200.0000, 0.0000, 50.0000, 50.0000, 500.0000),
('60000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000008', 150.0000, 10.0000, 30.0000, 100.0000, 500.0000),
('60000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 500.0000, 0.0000, 15.0000, 200.0000, 1000.0000),
-- Shanghai DC (6 items)
('60000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 40.0000, 2.0000, 7000.0000, 10.0000, 100.0000),
('60000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000002', 25.0000, 3.0000, 6000.0000, 10.0000, 80.0000),
('60000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000005', 15.0000, 0.0000, 8000.0000, 5.0000, 40.0000),
('60000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000011', 12.0000, 0.0000, 9000.0000, 5.0000, 30.0000),
('60000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000009', 80.0000, 5.0000, 80.0000, 50.0000, 200.0000),
('60000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000012', 60.0000, 0.0000, 80.0000, 30.0000, 150.0000),
-- Shenzhen Warehouse (6 items)
('60000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000003', 200.0000, 20.0000, 3500.0000, 15.0000, 300.0000),
('60000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000007', 500.0000, 0.0000, 50.0000, 50.0000, 1000.0000),
('60000000-0000-0000-0000-000000000023', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000006', 100.0000, 0.0000, 1200.0000, 20.0000, 200.0000),
('60000000-0000-0000-0000-000000000024', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000013', 50.0000, 0.0000, 100.0000, 30.0000, 100.0000),
('60000000-0000-0000-0000-000000000025', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000014', 200.0000, 0.0000, 10.0000, 100.0000, 500.0000),
('60000000-0000-0000-0000-000000000026', '00000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000015', 800.0000, 0.0000, 5.0000, 500.0000, 2000.0000)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 5: STOCK BATCHES (10 total)
-- ============================================================================

INSERT INTO stock_batches (id, inventory_item_id, batch_number, production_date, expiry_date, quantity, unit_cost, consumed) VALUES
-- iPhone 15 batches
('61000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 'BATCH-IP15-001', NOW() - INTERVAL '60 days', NULL, 25.0000, 7000.0000, false),
('61000000-0000-0000-0000-000000000002', '60000000-0000-0000-0000-000000000001', 'BATCH-IP15-002', NOW() - INTERVAL '30 days', NULL, 30.0000, 7000.0000, false),
-- Samsung S24 batches
('61000000-0000-0000-0000-000000000003', '60000000-0000-0000-0000-000000000002', 'BATCH-SAM24-001', NOW() - INTERVAL '45 days', NULL, 33.0000, 6000.0000, false),
-- Xiaomi 14 batches
('61000000-0000-0000-0000-000000000004', '60000000-0000-0000-0000-000000000003', 'BATCH-XM14-001', NOW() - INTERVAL '75 days', NULL, 50.0000, 3500.0000, false),
('61000000-0000-0000-0000-000000000005', '60000000-0000-0000-0000-000000000003', 'BATCH-XM14-002', NOW() - INTERVAL '40 days', NULL, 60.0000, 3500.0000, false),
-- MacBook Pro batches
('61000000-0000-0000-0000-000000000006', '60000000-0000-0000-0000-000000000004', 'BATCH-MAC-001', NOW() - INTERVAL '50 days', NULL, 22.0000, 12000.0000, false),
-- AirPods batches
('61000000-0000-0000-0000-000000000007', '60000000-0000-0000-0000-000000000005', 'BATCH-AIR-001', NOW() - INTERVAL '35 days', NULL, 85.0000, 1200.0000, false),
-- Shanghai iPhone batches
('61000000-0000-0000-0000-000000000008', '60000000-0000-0000-0000-000000000011', 'BATCH-IP15-SH01', NOW() - INTERVAL '25 days', NULL, 42.0000, 7000.0000, false),
-- Shenzhen Xiaomi batches
('61000000-0000-0000-0000-000000000009', '60000000-0000-0000-0000-000000000021', 'BATCH-XM14-SZ01', NOW() - INTERVAL '20 days', NULL, 100.0000, 3500.0000, false),
('61000000-0000-0000-0000-000000000010', '60000000-0000-0000-0000-000000000021', 'BATCH-XM14-SZ02', NOW() - INTERVAL '10 days', NULL, 120.0000, 3500.0000, false)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 5: STOCK LOCKS (5 total - linked to sales orders)
-- ============================================================================

INSERT INTO stock_locks (id, inventory_item_id, quantity, source_type, source_id, expire_at, released, consumed) VALUES
('62000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', 5.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000003', NOW() + INTERVAL '7 days', false, false),
('62000000-0000-0000-0000-000000000002', '60000000-0000-0000-0000-000000000003', 10.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000004', NOW() + INTERVAL '7 days', false, false),
('62000000-0000-0000-0000-000000000003', '60000000-0000-0000-0000-000000000002', 3.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000003', NOW() + INTERVAL '7 days', false, false),
('62000000-0000-0000-0000-000000000004', '60000000-0000-0000-0000-000000000004', 2.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000004', NOW() + INTERVAL '7 days', false, false),
('62000000-0000-0000-0000-000000000005', '60000000-0000-0000-0000-000000000007', 10.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000005', NOW() + INTERVAL '7 days', false, false)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 5: INVENTORY TRANSACTIONS (15 total)
-- ============================================================================

INSERT INTO inventory_transactions (id, tenant_id, inventory_item_id, warehouse_id, product_id, transaction_type, quantity, unit_cost, total_cost, balance_before, balance_after, source_type, source_id, reason) VALUES
-- Initial stock entries
('90000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'INBOUND', 55.0000, 7000.0000, 385000.0000, 0.0000, 55.0000, 'INITIAL_STOCK', 'INIT-IP15-001', 'Initial stock entry'),
('90000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000003', 'INBOUND', 110.0000, 3500.0000, 385000.0000, 0.0000, 110.0000, 'INITIAL_STOCK', 'INIT-XM14-001', 'Initial stock entry'),
('90000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000002', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000002', 'INBOUND', 33.0000, 6000.0000, 198000.0000, 0.0000, 33.0000, 'INITIAL_STOCK', 'INIT-SAM-001', 'Initial Samsung stock'),
('90000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000004', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000004', 'INBOUND', 22.0000, 12000.0000, 264000.0000, 0.0000, 22.0000, 'INITIAL_STOCK', 'INIT-MAC-001', 'Initial MacBook stock'),
('90000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000005', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000006', 'INBOUND', 85.0000, 1200.0000, 102000.0000, 0.0000, 85.0000, 'INITIAL_STOCK', 'INIT-AIR-001', 'Initial AirPods stock'),
('90000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000006', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000007', 'INBOUND', 200.0000, 50.0000, 10000.0000, 0.0000, 200.0000, 'INITIAL_STOCK', 'INIT-CHG-001', 'Initial charger stock'),
-- Purchase order receipts
('90000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000011', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 'INBOUND', 42.0000, 7000.0000, 294000.0000, 0.0000, 42.0000, 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000001', 'PO receipt'),
('90000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000021', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000003', 'INBOUND', 220.0000, 3500.0000, 770000.0000, 0.0000, 220.0000, 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000002', 'PO receipt'),
-- Sales order outbound
('90000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000011', '52000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 'OUTBOUND', 2.0000, 7000.0000, 14000.0000, 42.0000, 40.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000007', 'Sales shipment'),
('90000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000003', 'OUTBOUND', 5.0000, 3500.0000, 17500.0000, 110.0000, 105.0000, 'SALES_ORDER', '70000000-0000-0000-0000-000000000008', 'Sales shipment'),
-- Inventory adjustments
('90000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000003', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000003', 'ADJUSTMENT_DECREASE', 5.0000, 3500.0000, 17500.0000, 105.0000, 100.0000, 'STOCK_TAKING', 'ST-2026-001', 'Inventory count adjustment'),
-- Transfer between warehouses
('90000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000022', '52000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000007', 'TRANSFER_IN', 300.0000, 50.0000, 15000.0000, 200.0000, 500.0000, 'TRANSFER', 'TRF-2026-001', 'Transfer from Beijing'),
-- Returns processing
('90000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000001', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'RETURN', 1.0000, 7000.0000, 7000.0000, 54.0000, 55.0000, 'SALES_RETURN', '72000000-0000-0000-0000-000000000001', 'Customer return'),
('90000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000007', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000008', 'INBOUND', 160.0000, 30.0000, 4800.0000, 0.0000, 160.0000, 'INITIAL_STOCK', 'INIT-TSH-001', 'Initial T-shirt stock'),
('90000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000001', '60000000-0000-0000-0000-000000000008', '52000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000010', 'INBOUND', 500.0000, 15.0000, 7500.0000, 0.0000, 500.0000, 'INITIAL_STOCK', 'INIT-PAP-001', 'Initial A4 paper stock')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: SALES ORDERS (10 total with various statuses)
-- Status: DRAFT(2), CONFIRMED(2), SHIPPED(3), COMPLETED(2), CANCELLED(1)
-- ============================================================================

INSERT INTO sales_orders (id, tenant_id, order_number, customer_id, customer_name, warehouse_id, total_amount, discount_amount, payable_amount, status, remark, confirmed_at, shipped_at, completed_at, cancelled_at, cancel_reason) VALUES
-- DRAFT orders (2)
('70000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'SO-2026-0001', '50000000-0000-0000-0000-000000000001', 'Beijing Tech Solutions Ltd', '52000000-0000-0000-0000-000000000001', 17998.0000, 899.90, 17098.10, 'DRAFT', 'Pending customer confirmation', NULL, NULL, NULL, NULL, NULL),
('70000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'SO-2026-0002', '50000000-0000-0000-0000-000000000004', 'Chen Xiaoming', '52000000-0000-0000-0000-000000000001', 4999.0000, 0.00, 4999.00, 'DRAFT', 'New order', NULL, NULL, NULL, NULL, NULL),
-- CONFIRMED orders (2) - locked stock
('70000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'SO-2026-0003', '50000000-0000-0000-0000-000000000002', 'Shanghai Digital Corp', '52000000-0000-0000-0000-000000000001', 68991.0000, 5519.28, 63471.72, 'CONFIRMED', 'VIP customer order', NOW() - INTERVAL '3 days', NULL, NULL, NULL, NULL),
('70000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'SO-2026-0004', '50000000-0000-0000-0000-000000000006', 'Guangzhou Trading Co', '52000000-0000-0000-0000-000000000001', 79990.0000, 7999.00, 71991.00, 'CONFIRMED', 'Large bulk order', NOW() - INTERVAL '2 days', NULL, NULL, NULL, NULL),
-- SHIPPED orders (3)
('70000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'SO-2026-0005', '50000000-0000-0000-0000-000000000003', 'Shenzhen Hardware Inc', '52000000-0000-0000-0000-000000000001', 1580.0000, 47.40, 1532.60, 'SHIPPED', 'Express delivery', NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days', NULL, NULL, NULL),
('70000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'SO-2026-0006', '50000000-0000-0000-0000-000000000005', 'Wang Xiaohong', '52000000-0000-0000-0000-000000000002', 8999.0000, 899.90, 8099.10, 'SHIPPED', 'VIP shipping priority', NOW() - INTERVAL '4 days', NOW() - INTERVAL '3 days', NULL, NULL, NULL),
('70000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'SO-2026-0007', '50000000-0000-0000-0000-000000000008', 'Li Xiaofang', '52000000-0000-0000-0000-000000000002', 17998.0000, 539.94, 17458.06, 'SHIPPED', 'Standard shipping', NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days', NULL, NULL, NULL),
-- COMPLETED orders (2)
('70000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'SO-2026-0008', '50000000-0000-0000-0000-000000000009', 'Zhang Xiaoli', '52000000-0000-0000-0000-000000000001', 24995.0000, 1249.75, 23745.25, 'COMPLETED', 'Delivered and paid', NOW() - INTERVAL '10 days', NOW() - INTERVAL '8 days', NOW() - INTERVAL '5 days', NULL, NULL),
('70000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'SO-2026-0009', '50000000-0000-0000-0000-000000000010', 'Zhao Xiaojun', '52000000-0000-0000-0000-000000000003', 14999.0000, 1199.92, 13799.08, 'COMPLETED', 'Complete', NOW() - INTERVAL '15 days', NOW() - INTERVAL '12 days', NOW() - INTERVAL '8 days', NULL, NULL),
-- CANCELLED order (1)
('70000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'SO-2026-0010', '50000000-0000-0000-0000-000000000007', 'Hangzhou Software Ltd', '52000000-0000-0000-0000-000000000001', 10999.0000, 0.00, 10999.00, 'CANCELLED', 'Customer cancelled', NOW() - INTERVAL '7 days', NULL, NULL, NOW() - INTERVAL '6 days', 'Customer changed requirements')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: SALES ORDER ITEMS (25 total, 2-4 items per order)
-- ============================================================================

INSERT INTO sales_order_items (id, order_id, product_id, product_name, product_code, quantity, unit_price, amount, unit, conversion_rate, base_quantity, base_unit, remark) VALUES
-- SO-2026-0001 items (2 items)
('70100000-0000-0000-0000-000000000001', '70000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 2.0000, 8999.0000, 17998.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
-- SO-2026-0002 items (1 item)
('70100000-0000-0000-0000-000000000002', '70000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 1.0000, 4999.0000, 4999.0000, 'pcs', 1.0000, 1.0000, 'pcs', NULL),
-- SO-2026-0003 items (3 items)
('70100000-0000-0000-0000-000000000003', '70000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 5.0000, 8999.0000, 44995.0000, 'pcs', 1.0000, 5.0000, 'pcs', 'Bulk order'),
('70100000-0000-0000-0000-000000000004', '70000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000002', 'Samsung Galaxy S24', 'SAMSUNG24', 3.0000, 7999.0000, 23997.0000, 'pcs', 1.0000, 3.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000005', '70000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 1.0000, 99.0000, 99.0000, 'pcs', 1.0000, 1.0000, 'pcs', 'Free gift'),
-- SO-2026-0004 items (4 items)
('70100000-0000-0000-0000-000000000006', '70000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 10.0000, 4999.0000, 49990.0000, 'pcs', 1.0000, 10.0000, 'pcs', 'Reseller order'),
('70100000-0000-0000-0000-000000000007', '70000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000004', 'MacBook Pro 14', 'MACPRO', 2.0000, 14999.0000, 29998.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000008', '70000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 2.0000, 99.0000, 198.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000009', '70000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000012', 'USB-C Hub 7-in-1', 'USBHUB', 2.0000, 149.0000, 298.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
-- SO-2026-0005 items (3 items)
('70100000-0000-0000-0000-000000000010', '70000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000008', 'Mens Cotton T-Shirt', 'TSHIRT-M', 10.0000, 79.0000, 790.0000, 'pcs', 1.0000, 10.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000011', '70000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000009', 'Mens Denim Jeans', 'JEANS-M', 3.0000, 199.0000, 597.0000, 'pcs', 1.0000, 3.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000012', '70000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 2.0000, 99.0000, 198.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
-- SO-2026-0006 items (1 item)
('70100000-0000-0000-0000-000000000013', '70000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 1.0000, 8999.0000, 8999.0000, 'pcs', 1.0000, 1.0000, 'pcs', 'VIP customer'),
-- SO-2026-0007 items (2 items)
('70100000-0000-0000-0000-000000000014', '70000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 2.0000, 8999.0000, 17998.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000015', '70000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 1.0000, 1799.0000, 1799.0000, 'pcs', 1.0000, 1.0000, 'pcs', NULL),
-- SO-2026-0008 items (3 items)
('70100000-0000-0000-0000-000000000016', '70000000-0000-0000-0000-000000000008', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 5.0000, 4999.0000, 24995.0000, 'pcs', 1.0000, 5.0000, 'pcs', NULL),
-- SO-2026-0009 items (2 items)
('70100000-0000-0000-0000-000000000017', '70000000-0000-0000-0000-000000000009', '40000000-0000-0000-0000-000000000004', 'MacBook Pro 14', 'MACPRO', 1.0000, 14999.0000, 14999.0000, 'pcs', 1.0000, 1.0000, 'pcs', NULL),
-- SO-2026-0010 items (cancelled - 1 item)
('70100000-0000-0000-0000-000000000018', '70000000-0000-0000-0000-000000000010', '40000000-0000-0000-0000-000000000005', 'ThinkPad X1 Carbon', 'THINKPAD', 1.0000, 10999.0000, 10999.0000, 'pcs', 1.0000, 1.0000, 'pcs', 'Cancelled'),
-- Additional items for variety (7 more items)
('70100000-0000-0000-0000-000000000019', '70000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 2.0000, 1799.0000, 3598.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000020', '70000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000010', 'A4 Copy Paper', 'PAPER-A4', 5.0000, 29.0000, 145.0000, 'pack', 1.0000, 5.0000, 'pack', NULL),
('70100000-0000-0000-0000-000000000021', '70000000-0000-0000-0000-000000000008', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 5.0000, 99.0000, 495.0000, 'pcs', 1.0000, 5.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000022', '70000000-0000-0000-0000-000000000009', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 2.0000, 1799.0000, 3598.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000023', '70000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000012', 'USB-C Hub 7-in-1', 'USBHUB', 3.0000, 149.0000, 447.0000, 'pcs', 1.0000, 3.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000024', '70000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000008', 'Mens Cotton T-Shirt', 'TSHIRT-M', 5.0000, 79.0000, 395.0000, 'pcs', 1.0000, 5.0000, 'pcs', NULL),
('70100000-0000-0000-0000-000000000025', '70000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 2.0000, 99.0000, 198.0000, 'pcs', 1.0000, 2.0000, 'pcs', NULL)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: PURCHASE ORDERS (8 total with various statuses)
-- ============================================================================

INSERT INTO purchase_orders (id, tenant_id, order_number, supplier_id, supplier_name, warehouse_id, total_amount, discount_amount, payable_amount, status, remark, confirmed_at, completed_at, cancelled_at, cancel_reason) VALUES
-- DRAFT (1)
('71000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'PO-2026-0001', '51000000-0000-0000-0000-000000000001', 'Apple China Distribution', '52000000-0000-0000-0000-000000000001', 350000.0000, 0.00, 350000.00, 'DRAFT', 'Pending approval', NULL, NULL, NULL, NULL),
-- CONFIRMED (2)
('71000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'PO-2026-0002', '51000000-0000-0000-0000-000000000003', 'Xiaomi Technology Ltd', '52000000-0000-0000-0000-000000000003', 175000.0000, 0.00, 175000.00, 'CONFIRMED', 'Awaiting delivery', NOW() - INTERVAL '5 days', NULL, NULL, NULL),
('71000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'PO-2026-0003', '51000000-0000-0000-0000-000000000002', 'Samsung Electronics China', '52000000-0000-0000-0000-000000000002', 180000.0000, 0.00, 180000.00, 'CONFIRMED', 'In transit', NOW() - INTERVAL '3 days', NULL, NULL, NULL),
-- COMPLETED (4)
('71000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'PO-2026-0004', '51000000-0000-0000-0000-000000000001', 'Apple China Distribution', '52000000-0000-0000-0000-000000000002', 294000.0000, 0.00, 294000.00, 'COMPLETED', 'Fully received', NOW() - INTERVAL '20 days', NOW() - INTERVAL '15 days', NULL, NULL),
('71000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'PO-2026-0005', '51000000-0000-0000-0000-000000000003', 'Xiaomi Technology Ltd', '52000000-0000-0000-0000-000000000003', 770000.0000, 0.00, 770000.00, 'COMPLETED', 'Fully received', NOW() - INTERVAL '25 days', NOW() - INTERVAL '20 days', NULL, NULL),
('71000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'PO-2026-0006', '51000000-0000-0000-0000-000000000004', 'Lenovo Group China', '52000000-0000-0000-0000-000000000002', 120000.0000, 0.00, 120000.00, 'COMPLETED', 'Fully received', NOW() - INTERVAL '30 days', NOW() - INTERVAL '25 days', NULL, NULL),
('71000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'PO-2026-0007', '51000000-0000-0000-0000-000000000007', 'Fashion Textile Co', '52000000-0000-0000-0000-000000000001', 4800.0000, 0.00, 4800.00, 'COMPLETED', 'Clothing stock', NOW() - INTERVAL '15 days', NOW() - INTERVAL '10 days', NULL, NULL),
-- CANCELLED (1)
('71000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'PO-2026-0008', '51000000-0000-0000-0000-000000000006', 'Dell Technologies China', '52000000-0000-0000-0000-000000000001', 90000.0000, 0.00, 90000.00, 'CANCELLED', 'Price negotiation failed', NOW() - INTERVAL '10 days', NULL, NOW() - INTERVAL '8 days', 'Supplier price increase')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: PURCHASE ORDER ITEMS (20 total)
-- ============================================================================

INSERT INTO purchase_order_items (id, order_id, product_id, product_name, product_code, ordered_quantity, received_quantity, unit_cost, amount, unit, conversion_rate, base_quantity, base_unit, remark) VALUES
-- PO-2026-0001 items
('71100000-0000-0000-0000-000000000001', '71000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 50.0000, 0.0000, 7000.0000, 350000.0000, 'pcs', 1.0000, 50.0000, 'pcs', NULL),
-- PO-2026-0002 items
('71100000-0000-0000-0000-000000000002', '71000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 50.0000, 0.0000, 3500.0000, 175000.0000, 'pcs', 1.0000, 50.0000, 'pcs', NULL),
-- PO-2026-0003 items
('71100000-0000-0000-0000-000000000003', '71000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000002', 'Samsung Galaxy S24', 'SAMSUNG24', 30.0000, 0.0000, 6000.0000, 180000.0000, 'pcs', 1.0000, 30.0000, 'pcs', NULL),
-- PO-2026-0004 items (completed)
('71100000-0000-0000-0000-000000000004', '71000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 42.0000, 42.0000, 7000.0000, 294000.0000, 'pcs', 1.0000, 42.0000, 'pcs', 'Fully received'),
-- PO-2026-0005 items (completed)
('71100000-0000-0000-0000-000000000005', '71000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 220.0000, 220.0000, 3500.0000, 770000.0000, 'pcs', 1.0000, 220.0000, 'pcs', 'Fully received'),
-- PO-2026-0006 items (completed)
('71100000-0000-0000-0000-000000000006', '71000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000005', 'ThinkPad X1 Carbon', 'THINKPAD', 15.0000, 15.0000, 8000.0000, 120000.0000, 'pcs', 1.0000, 15.0000, 'pcs', 'Fully received'),
-- PO-2026-0007 items (completed - clothing)
('71100000-0000-0000-0000-000000000007', '71000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000008', 'Mens Cotton T-Shirt', 'TSHIRT-M', 160.0000, 160.0000, 30.0000, 4800.0000, 'pcs', 1.0000, 160.0000, 'pcs', 'Fully received'),
-- PO-2026-0008 items (cancelled)
('71100000-0000-0000-0000-000000000008', '71000000-0000-0000-0000-000000000008', '40000000-0000-0000-0000-000000000011', 'Dell XPS 15', 'DELLXPS', 10.0000, 0.0000, 9000.0000, 90000.0000, 'pcs', 1.0000, 10.0000, 'pcs', 'Cancelled'),
-- Additional items for variety
('71100000-0000-0000-0000-000000000009', '71000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 50.0000, 0.0000, 1200.0000, 60000.0000, 'pcs', 1.0000, 50.0000, 'pcs', NULL),
('71100000-0000-0000-0000-000000000010', '71000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 100.0000, 0.0000, 50.0000, 5000.0000, 'pcs', 1.0000, 100.0000, 'pcs', NULL),
('71100000-0000-0000-0000-000000000011', '71000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000012', 'USB-C Hub 7-in-1', 'USBHUB', 50.0000, 0.0000, 80.0000, 4000.0000, 'pcs', 1.0000, 50.0000, 'pcs', NULL),
('71100000-0000-0000-0000-000000000012', '71000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 30.0000, 30.0000, 1200.0000, 36000.0000, 'pcs', 1.0000, 30.0000, 'pcs', 'Received'),
('71100000-0000-0000-0000-000000000013', '71000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 200.0000, 200.0000, 50.0000, 10000.0000, 'pcs', 1.0000, 200.0000, 'pcs', 'Received'),
('71100000-0000-0000-0000-000000000014', '71000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000012', 'USB-C Hub 7-in-1', 'USBHUB', 60.0000, 60.0000, 80.0000, 4800.0000, 'pcs', 1.0000, 60.0000, 'pcs', 'Received'),
('71100000-0000-0000-0000-000000000015', '71000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000009', 'Mens Denim Jeans', 'JEANS-M', 50.0000, 50.0000, 80.0000, 4000.0000, 'pcs', 1.0000, 50.0000, 'pcs', 'Received'),
('71100000-0000-0000-0000-000000000016', '71000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000004', 'MacBook Pro 14', 'MACPRO', 10.0000, 0.0000, 12000.0000, 120000.0000, 'pcs', 1.0000, 10.0000, 'pcs', NULL),
('71100000-0000-0000-0000-000000000017', '71000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000012', 'USB-C Hub 7-in-1', 'USBHUB', 80.0000, 0.0000, 80.0000, 6400.0000, 'pcs', 1.0000, 80.0000, 'pcs', NULL),
('71100000-0000-0000-0000-000000000018', '71000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 40.0000, 0.0000, 1200.0000, 48000.0000, 'pcs', 1.0000, 40.0000, 'pcs', NULL),
('71100000-0000-0000-0000-000000000019', '71000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000011', 'Dell XPS 15', 'DELLXPS', 12.0000, 12.0000, 9000.0000, 108000.0000, 'pcs', 1.0000, 12.0000, 'pcs', 'Received'),
('71100000-0000-0000-0000-000000000020', '71000000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000013', 'Womens Summer Dress', 'DRESS-W', 50.0000, 50.0000, 100.0000, 5000.0000, 'pcs', 1.0000, 50.0000, 'pcs', 'Received')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: SALES RETURNS (5 total - 1 draft, 2 pending approval, 2 completed)
-- ============================================================================

INSERT INTO sales_returns (id, tenant_id, return_number, sales_order_id, sales_order_number, customer_id, customer_name, warehouse_id, total_refund, status, reason, remark, submitted_at, approved_at, approved_by, completed_at) VALUES
-- Draft (1)
('72000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'SR-2026-0005', '70000000-0000-0000-0000-000000000008', 'SO-2026-0008', '50000000-0000-0000-0000-000000000009', 'Zhang Xiaoli', '52000000-0000-0000-0000-000000000001', 299.0000, 'DRAFT', 'Customer request', 'Draft return for testing', NULL, NULL, NULL, NULL),
-- Pending approval (2)
('72000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'SR-2026-0001', '70000000-0000-0000-0000-000000000008', 'SO-2026-0008', '50000000-0000-0000-0000-000000000009', 'Zhang Xiaoli', '52000000-0000-0000-0000-000000000001', 4999.0000, 'PENDING', 'Product defect', 'Screen issue on one unit', NOW() - INTERVAL '2 days', NULL, NULL, NULL),
('72000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'SR-2026-0002', '70000000-0000-0000-0000-000000000009', 'SO-2026-0009', '50000000-0000-0000-0000-000000000010', 'Zhao Xiaojun', '52000000-0000-0000-0000-000000000003', 1799.0000, 'PENDING', 'Wrong item shipped', 'Customer ordered AirPods Pro, received regular AirPods', NOW() - INTERVAL '1 day', NULL, NULL, NULL),
-- Completed (2)
('72000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'SR-2026-0003', '70000000-0000-0000-0000-000000000008', 'SO-2026-0008', '50000000-0000-0000-0000-000000000009', 'Zhang Xiaoli', '52000000-0000-0000-0000-000000000001', 99.0000, 'COMPLETED', 'Change of mind', 'Customer no longer needs charger', NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days', '00000000-0000-0000-0000-000000000002', NOW() - INTERVAL '3 days'),
('72000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'SR-2026-0004', '70000000-0000-0000-0000-000000000005', 'SO-2026-0005', '50000000-0000-0000-0000-000000000003', 'Shenzhen Hardware Inc', '52000000-0000-0000-0000-000000000001', 79.0000, 'COMPLETED', 'Size exchange', 'T-shirt size too small', NOW() - INTERVAL '4 days', NOW() - INTERVAL '3 days', '00000000-0000-0000-0000-000000000002', NOW() - INTERVAL '2 days')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: SALES RETURN ITEMS (8 total)
-- ============================================================================

INSERT INTO sales_return_items (id, return_id, sales_order_item_id, product_id, product_name, product_code, original_quantity, return_quantity, unit_price, refund_amount, unit, reason, condition_on_return, conversion_rate, base_quantity, base_unit) VALUES
-- SR-2026-0001 items
('72100000-0000-0000-0000-000000000001', '72000000-0000-0000-0000-000000000001', '70100000-0000-0000-0000-000000000016', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 5.0000, 1.0000, 4999.0000, 4999.0000, 'pcs', 'Screen defect', 'unopened', 1.0000, 1.0000, 'pcs'),
-- SR-2026-0002 items
('72100000-0000-0000-0000-000000000002', '72000000-0000-0000-0000-000000000002', '70100000-0000-0000-0000-000000000022', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 2.0000, 1.0000, 1799.0000, 1799.0000, 'pcs', 'Wrong item', 'unopened', 1.0000, 1.0000, 'pcs'),
-- SR-2026-0003 items
('72100000-0000-0000-0000-000000000003', '72000000-0000-0000-0000-000000000003', '70100000-0000-0000-0000-000000000021', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 5.0000, 1.0000, 99.0000, 99.0000, 'pcs', 'Not needed', 'unopened', 1.0000, 1.0000, 'pcs'),
-- SR-2026-0004 items
('72100000-0000-0000-0000-000000000004', '72000000-0000-0000-0000-000000000004', '70100000-0000-0000-0000-000000000010', '40000000-0000-0000-0000-000000000008', 'Mens Cotton T-Shirt', 'TSHIRT-M', 10.0000, 1.0000, 79.0000, 79.0000, 'pcs', 'Wrong size', 'good', 1.0000, 1.0000, 'pcs'),
-- Additional return items
('72100000-0000-0000-0000-000000000005', '72000000-0000-0000-0000-000000000001', '70100000-0000-0000-0000-000000000021', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 5.0000, 2.0000, 99.0000, 198.0000, 'pcs', 'Defective', 'defective', 1.0000, 2.0000, 'pcs'),
('72100000-0000-0000-0000-000000000006', '72000000-0000-0000-0000-000000000002', '70100000-0000-0000-0000-000000000017', '40000000-0000-0000-0000-000000000004', 'MacBook Pro 14', 'MACPRO', 1.0000, 1.0000, 14999.0000, 14999.0000, 'pcs', 'DOA', 'defective', 1.0000, 1.0000, 'pcs'),
('72100000-0000-0000-0000-000000000007', '72000000-0000-0000-0000-000000000003', '70100000-0000-0000-0000-000000000016', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 5.0000, 1.0000, 4999.0000, 4999.0000, 'pcs', 'Change of mind', 'good', 1.0000, 1.0000, 'pcs'),
('72100000-0000-0000-0000-000000000008', '72000000-0000-0000-0000-000000000004', '70100000-0000-0000-0000-000000000011', '40000000-0000-0000-0000-000000000009', 'Mens Denim Jeans', 'JEANS-M', 3.0000, 1.0000, 199.0000, 199.0000, 'pcs', 'Wrong size', 'good', 1.0000, 1.0000, 'pcs')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: PURCHASE RETURNS (3 total)
-- ============================================================================

INSERT INTO purchase_returns (id, tenant_id, return_number, purchase_order_id, purchase_order_number, supplier_id, supplier_name, warehouse_id, total_refund, status, reason, remark, submitted_at, approved_at, approved_by, shipped_at, completed_at) VALUES
-- Pending (1)
('73000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'PR-2026-0001', '71000000-0000-0000-0000-000000000004', 'PO-2026-0004', '51000000-0000-0000-0000-000000000001', 'Apple China Distribution', '52000000-0000-0000-0000-000000000002', 14000.0000, 'PENDING', 'Defective units found', 'Two iPhones with screen issues', NOW() - INTERVAL '3 days', NULL, NULL, NULL, NULL),
-- Approved (1)
('73000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'PR-2026-0002', '71000000-0000-0000-0000-000000000005', 'PO-2026-0005', '51000000-0000-0000-0000-000000000003', 'Xiaomi Technology Ltd', '52000000-0000-0000-0000-000000000003', 7000.0000, 'APPROVED', 'Quality inspection failed', 'Two units failed QC', NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days', '00000000-0000-0000-0000-000000000002', NULL, NULL),
-- Completed (1)
('73000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'PR-2026-0003', '71000000-0000-0000-0000-000000000007', 'PO-2026-0007', '51000000-0000-0000-0000-000000000007', 'Fashion Textile Co', '52000000-0000-0000-0000-000000000001', 240.0000, 'COMPLETED', 'Color mismatch', 'Wrong color batch received', NOW() - INTERVAL '8 days', NOW() - INTERVAL '7 days', '00000000-0000-0000-0000-000000000002', NOW() - INTERVAL '6 days', NOW() - INTERVAL '4 days')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 6: PURCHASE RETURN ITEMS (6 total)
-- ============================================================================

INSERT INTO purchase_return_items (id, return_id, purchase_order_item_id, product_id, product_name, product_code, original_quantity, return_quantity, unit_cost, refund_amount, unit, reason, condition_on_return) VALUES
-- PR-2026-0001 items
('73100000-0000-0000-0000-000000000001', '73000000-0000-0000-0000-000000000001', '71100000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000001', 'iPhone 15 Pro', 'IPHONE15', 42.0000, 2.0000, 7000.0000, 14000.0000, 'pcs', 'Screen defect', 'defective'),
-- PR-2026-0002 items
('73100000-0000-0000-0000-000000000002', '73000000-0000-0000-0000-000000000002', '71100000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'Xiaomi 14 Pro', 'XIAOMI14', 220.0000, 2.0000, 3500.0000, 7000.0000, 'pcs', 'QC failed', 'defective'),
-- PR-2026-0003 items
('73100000-0000-0000-0000-000000000003', '73000000-0000-0000-0000-000000000003', '71100000-0000-0000-0000-000000000007', '40000000-0000-0000-0000-000000000008', 'Mens Cotton T-Shirt', 'TSHIRT-M', 160.0000, 8.0000, 30.0000, 240.0000, 'pcs', 'Color mismatch', 'good'),
-- Additional return items
('73100000-0000-0000-0000-000000000004', '73000000-0000-0000-0000-000000000001', '71100000-0000-0000-0000-000000000012', '40000000-0000-0000-0000-000000000006', 'AirPods Pro', 'AIRPODS', 30.0000, 1.0000, 1200.0000, 1200.0000, 'pcs', 'DOA', 'defective'),
('73100000-0000-0000-0000-000000000005', '73000000-0000-0000-0000-000000000002', '71100000-0000-0000-0000-000000000013', '40000000-0000-0000-0000-000000000007', 'USB-C Charger 65W', 'CHARGER', 200.0000, 5.0000, 50.0000, 250.0000, 'pcs', 'No output', 'defective'),
('73100000-0000-0000-0000-000000000006', '73000000-0000-0000-0000-000000000003', '71100000-0000-0000-0000-000000000015', '40000000-0000-0000-0000-000000000009', 'Mens Denim Jeans', 'JEANS-M', 50.0000, 3.0000, 80.0000, 240.0000, 'pcs', 'Stitching defect', 'defective')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: ACCOUNT RECEIVABLES (8 total - linked to sales orders)
-- ============================================================================

INSERT INTO account_receivables (id, tenant_id, receivable_number, customer_id, customer_name, source_type, source_id, source_number, total_amount, paid_amount, outstanding_amount, status, due_date) VALUES
-- PENDING (3)
('80000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'AR-2026-0001', '50000000-0000-0000-0000-000000000002', 'Shanghai Digital Corp', 'SALES_ORDER', '70000000-0000-0000-0000-000000000003', 'SO-2026-0003', 63471.72, 0.00, 63471.72, 'PENDING', NOW() + INTERVAL '30 days'),
('80000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'AR-2026-0002', '50000000-0000-0000-0000-000000000006', 'Guangzhou Trading Co', 'SALES_ORDER', '70000000-0000-0000-0000-000000000004', 'SO-2026-0004', 71991.00, 0.00, 71991.00, 'PENDING', NOW() + INTERVAL '45 days'),
('80000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'AR-2026-0003', '50000000-0000-0000-0000-000000000003', 'Shenzhen Hardware Inc', 'SALES_ORDER', '70000000-0000-0000-0000-000000000005', 'SO-2026-0005', 1532.60, 0.00, 1532.60, 'PENDING', NOW() + INTERVAL '15 days'),
-- PARTIAL (2)
('80000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'AR-2026-0004', '50000000-0000-0000-0000-000000000005', 'Wang Xiaohong', 'SALES_ORDER', '70000000-0000-0000-0000-000000000006', 'SO-2026-0006', 8099.10, 5000.00, 3099.10, 'PARTIAL', NOW() + INTERVAL '20 days'),
('80000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'AR-2026-0005', '50000000-0000-0000-0000-000000000008', 'Li Xiaofang', 'SALES_ORDER', '70000000-0000-0000-0000-000000000007', 'SO-2026-0007', 17458.06, 10000.00, 7458.06, 'PARTIAL', NOW() + INTERVAL '25 days'),
-- PAID (3)
('80000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'AR-2026-0006', '50000000-0000-0000-0000-000000000009', 'Zhang Xiaoli', 'SALES_ORDER', '70000000-0000-0000-0000-000000000008', 'SO-2026-0008', 23745.25, 23745.25, 0.00, 'PAID', NOW() - INTERVAL '5 days'),
('80000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'AR-2026-0007', '50000000-0000-0000-0000-000000000010', 'Zhao Xiaojun', 'SALES_ORDER', '70000000-0000-0000-0000-000000000009', 'SO-2026-0009', 13799.08, 13799.08, 0.00, 'PAID', NOW() - INTERVAL '8 days'),
('80000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'AR-2026-0008', '50000000-0000-0000-0000-000000000001', 'Beijing Tech Solutions Ltd', 'MANUAL', '00000000-0000-0000-0000-000000000000', 'MANUAL-001', 5000.00, 5000.00, 0.00, 'PAID', NOW() - INTERVAL '10 days')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: ACCOUNT PAYABLES (6 total - linked to purchase orders)
-- ============================================================================

INSERT INTO account_payables (id, tenant_id, payable_number, supplier_id, supplier_name, source_type, source_id, source_number, total_amount, paid_amount, outstanding_amount, status, due_date) VALUES
-- PENDING (2)
('81000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'AP-2026-0001', '51000000-0000-0000-0000-000000000003', 'Xiaomi Technology Ltd', 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000002', 'PO-2026-0002', 175000.00, 0.00, 175000.00, 'PENDING', NOW() + INTERVAL '30 days'),
('81000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'AP-2026-0002', '51000000-0000-0000-0000-000000000002', 'Samsung Electronics China', 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000003', 'PO-2026-0003', 180000.00, 0.00, 180000.00, 'PENDING', NOW() + INTERVAL '45 days'),
-- PARTIAL (1)
('81000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'AP-2026-0003', '51000000-0000-0000-0000-000000000001', 'Apple China Distribution', 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000004', 'PO-2026-0004', 294000.00, 150000.00, 144000.00, 'PARTIAL', NOW() + INTERVAL '15 days'),
-- PAID (3)
('81000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'AP-2026-0004', '51000000-0000-0000-0000-000000000003', 'Xiaomi Technology Ltd', 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000005', 'PO-2026-0005', 770000.00, 770000.00, 0.00, 'PAID', NOW() - INTERVAL '15 days'),
('81000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'AP-2026-0005', '51000000-0000-0000-0000-000000000004', 'Lenovo Group China', 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000006', 'PO-2026-0006', 120000.00, 120000.00, 0.00, 'PAID', NOW() - INTERVAL '20 days'),
('81000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'AP-2026-0006', '51000000-0000-0000-0000-000000000007', 'Fashion Textile Co', 'PURCHASE_ORDER', '71000000-0000-0000-0000-000000000007', 'PO-2026-0007', 4800.00, 4800.00, 0.00, 'PAID', NOW() - INTERVAL '5 days')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: RECEIPT VOUCHERS (5 total - customer payments)
-- ============================================================================

INSERT INTO receipt_vouchers (id, tenant_id, voucher_number, customer_id, customer_name, amount, allocated_amount, unallocated_amount, payment_method, status, receipt_date, remark) VALUES
('82000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'RV-2026-0001', '50000000-0000-0000-0000-000000000005', 'Wang Xiaohong', 5000.00, 5000.00, 0.00, 'WECHAT', 'CONFIRMED', NOW() - INTERVAL '3 days', 'Partial payment for SO-2026-0006'),
('82000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'RV-2026-0002', '50000000-0000-0000-0000-000000000008', 'Li Xiaofang', 10000.00, 10000.00, 0.00, 'BANK_TRANSFER', 'CONFIRMED', NOW() - INTERVAL '2 days', 'Partial payment for SO-2026-0007'),
('82000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'RV-2026-0003', '50000000-0000-0000-0000-000000000009', 'Zhang Xiaoli', 23745.25, 23745.25, 0.00, 'ALIPAY', 'CONFIRMED', NOW() - INTERVAL '6 days', 'Full payment for SO-2026-0008'),
('82000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'RV-2026-0004', '50000000-0000-0000-0000-000000000010', 'Zhao Xiaojun', 13799.08, 13799.08, 0.00, 'BANK_TRANSFER', 'CONFIRMED', NOW() - INTERVAL '9 days', 'Full payment for SO-2026-0009'),
('82000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'RV-2026-0005', '50000000-0000-0000-0000-000000000001', 'Beijing Tech Solutions Ltd', 5000.00, 5000.00, 0.00, 'CASH', 'CONFIRMED', NOW() - INTERVAL '11 days', 'Manual receivable payment')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: PAYMENT VOUCHERS (4 total - supplier payments)
-- ============================================================================

INSERT INTO payment_vouchers (id, tenant_id, voucher_number, supplier_id, supplier_name, amount, allocated_amount, unallocated_amount, payment_method, status, payment_date, remark) VALUES
('83000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'PV-2026-0001', '51000000-0000-0000-0000-000000000001', 'Apple China Distribution', 150000.00, 150000.00, 0.00, 'BANK_TRANSFER', 'CONFIRMED', NOW() - INTERVAL '5 days', 'Partial payment for PO-2026-0004'),
('83000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'PV-2026-0002', '51000000-0000-0000-0000-000000000003', 'Xiaomi Technology Ltd', 770000.00, 770000.00, 0.00, 'BANK_TRANSFER', 'CONFIRMED', NOW() - INTERVAL '16 days', 'Full payment for PO-2026-0005'),
('83000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'PV-2026-0003', '51000000-0000-0000-0000-000000000004', 'Lenovo Group China', 120000.00, 120000.00, 0.00, 'BANK_TRANSFER', 'CONFIRMED', NOW() - INTERVAL '21 days', 'Full payment for PO-2026-0006'),
('83000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'PV-2026-0004', '51000000-0000-0000-0000-000000000007', 'Fashion Textile Co', 4800.00, 4800.00, 0.00, 'BANK_TRANSFER', 'CONFIRMED', NOW() - INTERVAL '6 days', 'Full payment for PO-2026-0007')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: EXPENSE RECORDS (6 total)
-- ============================================================================

INSERT INTO expense_records (id, tenant_id, expense_number, category, amount, incurred_at, description, status) VALUES
('84000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0001', 'RENT', 15000.00, NOW() - INTERVAL '15 days', 'Monthly warehouse rent - Beijing', 'APPROVED'),
('84000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0002', 'UTILITIES', 3500.00, NOW() - INTERVAL '10 days', 'Electricity and water bills', 'APPROVED'),
('84000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0003', 'OTHER', 8000.00, NOW() - INTERVAL '5 days', 'Shipping costs for January', 'APPROVED'),
('84000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0004', 'SALARY', 50000.00, NOW() - INTERVAL '1 day', 'Staff salaries', 'DRAFT'),
('84000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0005', 'RENT', 12000.00, NOW() - INTERVAL '45 days', 'Shanghai warehouse rent - December', 'APPROVED'),
('84000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'EXP-2026-0006', 'OFFICE', 2500.00, NOW() - INTERVAL '8 days', 'Office supplies and equipment', 'APPROVED')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: OTHER INCOME RECORDS (4 total)
-- ============================================================================

INSERT INTO other_income_records (id, tenant_id, income_number, category, amount, received_at, description, status) VALUES
('85000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'INC-2026-0001', 'OTHER', 5000.00, NOW() - INTERVAL '12 days', 'Installation service fee', 'CONFIRMED'),
('85000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'INC-2026-0002', 'INTEREST', 1200.00, NOW() - INTERVAL '8 days', 'Bank interest income', 'CONFIRMED'),
('85000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'INC-2026-0003', 'OTHER', 3000.00, NOW() - INTERVAL '20 days', 'Training service income', 'CONFIRMED'),
('85000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'INC-2026-0004', 'INTEREST', 800.00, NOW() - INTERVAL '38 days', 'December bank interest', 'CONFIRMED')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 7: BALANCE TRANSACTIONS (10 total - customer prepaid balance)
-- ============================================================================

INSERT INTO balance_transactions (id, tenant_id, customer_id, transaction_type, amount, balance_before, balance_after, source_type, source_id, remark) VALUES
-- Beijing Tech deposits and usage
('86000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000001', 'RECHARGE', 10000.00, 0.00, 10000.00, 'MANUAL', NULL, 'Initial deposit'),
('86000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000001', 'CONSUME', 5000.00, 10000.00, 5000.00, 'SALES_ORDER', '70000000-0000-0000-0000-000000000001', 'Payment for SO-2026-0001'),
-- Shanghai Digital deposits
('86000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000002', 'RECHARGE', 20000.00, 0.00, 20000.00, 'MANUAL', NULL, 'VIP customer deposit'),
('86000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000002', 'CONSUME', 10000.00, 20000.00, 10000.00, 'MANUAL', NULL, 'Partial balance usage'),
-- Individual customer transactions
('86000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000004', 'RECHARGE', 1000.00, 0.00, 1000.00, 'MANUAL', NULL, 'Small recharge'),
('86000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000004', 'CONSUME', 500.00, 1000.00, 500.00, 'MANUAL', NULL, 'Purchase usage'),
-- Wang Xiaohong VIP deposits
('86000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000005', 'RECHARGE', 5000.00, 0.00, 5000.00, 'MANUAL', NULL, 'VIP deposit'),
('86000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000005', 'CONSUME', 3000.00, 5000.00, 2000.00, 'MANUAL', NULL, 'Order payment'),
-- Guangzhou Trading Co
('86000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000006', 'RECHARGE', 20000.00, 0.00, 20000.00, 'MANUAL', NULL, 'Large enterprise deposit'),
('86000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000006', 'CONSUME', 5000.00, 20000.00, 15000.00, 'MANUAL', NULL, 'Partial order payment')
ON CONFLICT DO NOTHING;

-- ============================================================================
-- LAYER 8: FEATURE FLAGS (3 total for E2E testing)
-- ============================================================================

INSERT INTO feature_flags (id, key, name, description, type, status, default_value, rules, tags, version, created_by, updated_by) VALUES
-- Boolean flag (enabled)
('f0000000-0000-0000-0000-000000000001', 'test_boolean_flag', 'Test Boolean Flag', 'A test boolean flag for E2E testing - enabled by default', 'boolean', 'enabled', '{"enabled": true}', NULL, ARRAY['test', 'e2e', 'boolean'], 1, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002'),
-- Percentage flag (50%)
('f0000000-0000-0000-0000-000000000002', 'test_percentage_flag', 'Test Percentage Flag', 'A test percentage flag with 50% rollout', 'percentage', 'enabled', '{"enabled": true, "metadata": {"percentage": 50}}', NULL, ARRAY['test', 'e2e', 'percentage'], 1, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002'),
-- Variant flag (A/B/C)
('f0000000-0000-0000-0000-000000000003', 'test_variant_flag', 'Test Variant Flag', 'A test variant flag with A/B/C variants', 'variant', 'enabled', '{"enabled": true, "variant": "A", "metadata": {"variants": ["A", "B", "C"]}}', NULL, ARRAY['test', 'e2e', 'variant'], 1, '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002')
ON CONFLICT (key) DO NOTHING;

-- ============================================================================
-- SUMMARY
-- ============================================================================
-- Comprehensive seed data created:
--
-- LAYER 1 - Base Data:
--   - 3 tenants (default + alpha + beta)
--   - 10 categories (4 root + 6 sub)
--
-- LAYER 2 - Master Data:
--   - 15 customer levels (5 per tenant)
--   - 15 products (phones, computers, accessories, clothing, office, food)
--   - 10 customers (5 organizations + 5 individuals, various levels)
--   - 8 suppliers (manufacturers, distributors, wholesalers)
--   - 5 warehouses (3 physical + 1 virtual + 1 returns)
--
-- LAYER 3 - Users/Roles:
--   - 6 users (admin + sales/warehouse/finance/purchaser/cashier)
--   - Role permissions for SALES, WAREHOUSE, ACCOUNTANT roles
--
-- LAYER 4 - Product Units:
--   - 10 product units for multi-unit support (A4 paper, pens, T-shirts, chargers)
--
-- LAYER 5 - Inventory:
--   - 20 inventory items across 3 warehouses
--   - 10 stock batches with production dates
--   - 5 stock locks (linked to confirmed sales orders)
--   - 15 inventory transactions (initial, purchases, sales, adjustments, returns)
--
-- LAYER 6 - Trade:
--   - 10 sales orders (DRAFT 2, CONFIRMED 2, SHIPPED 3, COMPLETED 2, CANCELLED 1)
--   - 25 sales order items
--   - 8 purchase orders (DRAFT 1, CONFIRMED 2, COMPLETED 4, CANCELLED 1)
--   - 20 purchase order items
--   - 4 sales returns (2 pending, 2 completed)
--   - 8 sales return items
--   - 3 purchase returns (1 submitted, 1 approved, 1 completed)
--   - 6 purchase return items
--
-- LAYER 7 - Finance:
--   - 8 account receivables (3 pending, 2 partial, 3 paid)
--   - 6 account payables (2 pending, 1 partial, 3 paid)
--   - 5 receipt vouchers (customer payments)
--   - 4 payment vouchers (supplier payments)
--   - 6 expense records (rent, utilities, salary, other)
--   - 4 other income records (services, interest)
--   - 10 balance transactions (recharges and consumptions)
--
-- LAYER 8 - Feature Flags:
--   - 3 feature flags (test_boolean_flag, test_percentage_flag, test_variant_flag)
--
-- Data Consistency:
--   - Sales order amounts = SUM(order item amounts)
--   - Receivables linked to shipped/completed sales orders
--   - Payables linked to confirmed/completed purchase orders
--   - Stock locks linked to confirmed sales orders
--   - Inventory transactions match inventory item quantities
--   - Customer balances = SUM(recharges) - SUM(consumptions)

-- ============================================================================
-- LAYER 9: PRINT TEMPLATES (Default templates for all document types)
-- ============================================================================

INSERT INTO print_templates (id, tenant_id, document_type, name, description, content, paper_size, orientation, margin_top, margin_right, margin_bottom, margin_left, is_default, status, created_at, updated_at, version) VALUES
-- Sales Order Template (SALES_ORDER)
('a0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'SALES_ORDER', '销售订单-A4', '标准销售订单打印模板', '<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售订单 - {{ .Document.OrderNumber }}</title>
    <style>
        body { font-family: "Microsoft YaHei", sans-serif; font-size: 12px; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; border-bottom: 2px solid #333; padding-bottom: 10px; }
        .title { font-size: 20px; font-weight: bold; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 15px; }
        .info-left, .info-right { width: 48%; }
        .info-row { margin-bottom: 5px; }
        .info-label { font-weight: bold; display: inline-block; width: 80px; }
        table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        th, td { border: 1px solid #333; padding: 8px; text-align: center; }
        th { background: #f5f5f5; }
        .text-left { text-align: left; }
        .text-right { text-align: right; }
        .summary { margin-top: 15px; text-align: right; }
        .total { font-size: 16px; font-weight: bold; color: #c00; }
        .signature { display: flex; justify-content: space-around; margin-top: 30px; }
        .sign-box { text-align: center; width: 30%; }
        .sign-line { border-bottom: 1px solid #333; height: 30px; margin-bottom: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <div class="title">销 售 订 单</div>
    </div>
    <div class="info-section">
        <div class="info-left">
            <div class="info-row"><span class="info-label">订单编号:</span>{{ .Document.OrderNumber }}</div>
            <div class="info-row"><span class="info-label">客户名称:</span>{{ .Document.Customer.Name }}</div>
            <div class="info-row"><span class="info-label">联系人:</span>{{ .Document.Customer.Contact }}</div>
            <div class="info-row"><span class="info-label">联系电话:</span>{{ .Document.Customer.Phone }}</div>
        </div>
        <div class="info-right">
            <div class="info-row"><span class="info-label">订单日期:</span>{{ .Meta.CreatedAtFormatted }}</div>
            <div class="info-row"><span class="info-label">订单状态:</span>{{ .Meta.StatusText }}</div>
            <div class="info-row"><span class="info-label">打印日期:</span>{{ .PrintDate }}</div>
        </div>
    </div>
    <table>
        <thead>
            <tr>
                <th style="width:40px">序号</th>
                <th style="width:100px">商品编码</th>
                <th class="text-left">商品名称</th>
                <th style="width:50px">单位</th>
                <th style="width:70px">数量</th>
                <th style="width:80px">单价</th>
                <th style="width:90px">金额</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Document.Items }}
            <tr>
                <td>{{ .Index }}</td>
                <td>{{ .ProductCode }}</td>
                <td class="text-left">{{ .ProductName }}</td>
                <td>{{ .Unit }}</td>
                <td class="text-right">{{ .QuantityFormatted }}</td>
                <td class="text-right">{{ .UnitPriceFormatted }}</td>
                <td class="text-right">{{ .AmountFormatted }}</td>
            </tr>
            {{ end }}
        </tbody>
    </table>
    <div class="summary">
        <div>商品种类: {{ .Document.ItemCount }} 种</div>
        <div class="total">合计金额: {{ .Document.PayableAmountFormatted }}</div>
        <div>大写: {{ .Document.PayableAmountChinese }}</div>
    </div>
    {{ if .Document.Remark }}<div style="margin-top:15px;padding:10px;background:#fffef0;border:1px solid #e6e6c0;"><b>备注:</b> {{ .Document.Remark }}</div>{{ end }}
    <div class="signature">
        <div class="sign-box"><div class="sign-line"></div>制单人</div>
        <div class="sign-box"><div class="sign-line"></div>审核人</div>
        <div class="sign-box"><div class="sign-line"></div>客户签收</div>
    </div>
</body>
</html>', 'A4', 'PORTRAIT', 10, 10, 10, 10, true, 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1),

-- Purchase Order Template (PURCHASE_ORDER)
('a0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'PURCHASE_ORDER', '采购订单-A4', '标准采购订单打印模板', '<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>采购订单 - {{ .Document.OrderNumber }}</title>
    <style>
        body { font-family: "Microsoft YaHei", sans-serif; font-size: 12px; padding: 20px; }
        .header { text-align: center; margin-bottom: 20px; border-bottom: 2px solid #333; padding-bottom: 10px; }
        .title { font-size: 20px; font-weight: bold; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 15px; }
        .info-left, .info-right { width: 48%; }
        .info-row { margin-bottom: 5px; }
        .info-label { font-weight: bold; display: inline-block; width: 80px; }
        table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        th, td { border: 1px solid #333; padding: 8px; text-align: center; }
        th { background: #f5f5f5; }
        .text-left { text-align: left; }
        .text-right { text-align: right; }
        .summary { margin-top: 15px; text-align: right; }
        .total { font-size: 16px; font-weight: bold; color: #c00; }
        .signature { display: flex; justify-content: space-around; margin-top: 30px; }
        .sign-box { text-align: center; width: 30%; }
        .sign-line { border-bottom: 1px solid #333; height: 30px; margin-bottom: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <div class="title">采 购 订 单</div>
    </div>
    <div class="info-section">
        <div class="info-left">
            <div class="info-row"><span class="info-label">订单编号:</span>{{ .Document.OrderNumber }}</div>
            <div class="info-row"><span class="info-label">供应商:</span>{{ .Document.Supplier.Name }}</div>
            <div class="info-row"><span class="info-label">联系人:</span>{{ .Document.Supplier.Contact }}</div>
            <div class="info-row"><span class="info-label">联系电话:</span>{{ .Document.Supplier.Phone }}</div>
        </div>
        <div class="info-right">
            <div class="info-row"><span class="info-label">订单日期:</span>{{ .Meta.CreatedAtFormatted }}</div>
            <div class="info-row"><span class="info-label">订单状态:</span>{{ .Meta.StatusText }}</div>
            <div class="info-row"><span class="info-label">打印日期:</span>{{ .PrintDate }}</div>
        </div>
    </div>
    <table>
        <thead>
            <tr>
                <th style="width:40px">序号</th>
                <th style="width:100px">商品编码</th>
                <th class="text-left">商品名称</th>
                <th style="width:50px">单位</th>
                <th style="width:70px">数量</th>
                <th style="width:80px">单价</th>
                <th style="width:90px">金额</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Document.Items }}
            <tr>
                <td>{{ .Index }}</td>
                <td>{{ .ProductCode }}</td>
                <td class="text-left">{{ .ProductName }}</td>
                <td>{{ .Unit }}</td>
                <td class="text-right">{{ .QuantityFormatted }}</td>
                <td class="text-right">{{ .UnitPriceFormatted }}</td>
                <td class="text-right">{{ .AmountFormatted }}</td>
            </tr>
            {{ end }}
        </tbody>
    </table>
    <div class="summary">
        <div>商品种类: {{ .Document.ItemCount }} 种</div>
        <div class="total">合计金额: {{ .Document.TotalAmountFormatted }}</div>
        <div>大写: {{ .Document.TotalAmountChinese }}</div>
    </div>
    {{ if .Document.Remark }}<div style="margin-top:15px;padding:10px;background:#fffef0;border:1px solid #e6e6c0;"><b>备注:</b> {{ .Document.Remark }}</div>{{ end }}
    <div class="signature">
        <div class="sign-box"><div class="sign-line"></div>制单人</div>
        <div class="sign-box"><div class="sign-line"></div>审核人</div>
        <div class="sign-box"><div class="sign-line"></div>供应商确认</div>
    </div>
</body>
</html>', 'A4', 'PORTRAIT', 10, 10, 10, 10, true, 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1),

-- Sales Order A5 Template
('a0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'SALES_ORDER', '销售订单-A5', 'A5尺寸销售订单模板', '<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售订单 - {{ .Document.OrderNumber }}</title>
    <style>
        body { font-family: "Microsoft YaHei", sans-serif; font-size: 10px; padding: 10px; }
        .header { text-align: center; margin-bottom: 10px; border-bottom: 1px solid #333; padding-bottom: 5px; }
        .title { font-size: 14px; font-weight: bold; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 9px; }
        .info-left, .info-right { width: 48%; }
        .info-row { margin-bottom: 3px; }
        .info-label { font-weight: bold; display: inline-block; width: 60px; }
        table { width: 100%; border-collapse: collapse; margin: 8px 0; font-size: 9px; }
        th, td { border: 1px solid #333; padding: 4px; text-align: center; }
        th { background: #f5f5f5; }
        .text-left { text-align: left; }
        .text-right { text-align: right; }
        .summary { margin-top: 8px; text-align: right; font-size: 9px; }
        .total { font-size: 12px; font-weight: bold; color: #c00; }
        .signature { display: flex; justify-content: space-around; margin-top: 15px; font-size: 9px; }
        .sign-box { text-align: center; width: 30%; }
        .sign-line { border-bottom: 1px solid #333; height: 20px; margin-bottom: 3px; }
    </style>
</head>
<body>
    <div class="header"><div class="title">销 售 订 单</div></div>
    <div class="info-section">
        <div class="info-left">
            <div class="info-row"><span class="info-label">订单编号:</span>{{ .Document.OrderNumber }}</div>
            <div class="info-row"><span class="info-label">客户:</span>{{ .Document.Customer.Name }}</div>
            <div class="info-row"><span class="info-label">电话:</span>{{ .Document.Customer.Phone }}</div>
        </div>
        <div class="info-right">
            <div class="info-row"><span class="info-label">日期:</span>{{ .Meta.CreatedAtFormatted }}</div>
            <div class="info-row"><span class="info-label">状态:</span>{{ .Meta.StatusText }}</div>
        </div>
    </div>
    <table>
        <thead><tr><th style="width:25px">序</th><th class="text-left">商品</th><th style="width:35px">单位</th><th style="width:45px">数量</th><th style="width:55px">单价</th><th style="width:60px">金额</th></tr></thead>
        <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td class="text-left">{{ .ProductName }}</td><td>{{ .Unit }}</td><td class="text-right">{{ .QuantityFormatted }}</td><td class="text-right">{{ .UnitPriceFormatted }}</td><td class="text-right">{{ .AmountFormatted }}</td></tr>{{ end }}</tbody>
    </table>
    <div class="summary"><div class="total">合计: {{ .Document.PayableAmountFormatted }}</div></div>
    <div class="signature"><div class="sign-box"><div class="sign-line"></div>制单</div><div class="sign-box"><div class="sign-line"></div>审核</div><div class="sign-box"><div class="sign-line"></div>签收</div></div>
</body>
</html>', 'A5', 'PORTRAIT', 5, 5, 5, 5, false, 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1),

-- Purchase Order A5 Template
('a0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'PURCHASE_ORDER', '采购订单-A5', 'A5尺寸采购订单模板', '<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>采购订单 - {{ .Document.OrderNumber }}</title>
    <style>
        body { font-family: "Microsoft YaHei", sans-serif; font-size: 10px; padding: 10px; }
        .header { text-align: center; margin-bottom: 10px; border-bottom: 1px solid #333; padding-bottom: 5px; }
        .title { font-size: 14px; font-weight: bold; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 9px; }
        .info-left, .info-right { width: 48%; }
        .info-row { margin-bottom: 3px; }
        .info-label { font-weight: bold; display: inline-block; width: 60px; }
        table { width: 100%; border-collapse: collapse; margin: 8px 0; font-size: 9px; }
        th, td { border: 1px solid #333; padding: 4px; text-align: center; }
        th { background: #f5f5f5; }
        .text-left { text-align: left; }
        .text-right { text-align: right; }
        .summary { margin-top: 8px; text-align: right; font-size: 9px; }
        .total { font-size: 12px; font-weight: bold; color: #c00; }
        .signature { display: flex; justify-content: space-around; margin-top: 15px; font-size: 9px; }
        .sign-box { text-align: center; width: 30%; }
        .sign-line { border-bottom: 1px solid #333; height: 20px; margin-bottom: 3px; }
    </style>
</head>
<body>
    <div class="header"><div class="title">采 购 订 单</div></div>
    <div class="info-section">
        <div class="info-left">
            <div class="info-row"><span class="info-label">订单编号:</span>{{ .Document.OrderNumber }}</div>
            <div class="info-row"><span class="info-label">供应商:</span>{{ .Document.Supplier.Name }}</div>
            <div class="info-row"><span class="info-label">电话:</span>{{ .Document.Supplier.Phone }}</div>
        </div>
        <div class="info-right">
            <div class="info-row"><span class="info-label">日期:</span>{{ .Meta.CreatedAtFormatted }}</div>
            <div class="info-row"><span class="info-label">状态:</span>{{ .Meta.StatusText }}</div>
        </div>
    </div>
    <table>
        <thead><tr><th style="width:25px">序</th><th class="text-left">商品</th><th style="width:35px">单位</th><th style="width:45px">数量</th><th style="width:55px">单价</th><th style="width:60px">金额</th></tr></thead>
        <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td class="text-left">{{ .ProductName }}</td><td>{{ .Unit }}</td><td class="text-right">{{ .QuantityFormatted }}</td><td class="text-right">{{ .UnitPriceFormatted }}</td><td class="text-right">{{ .AmountFormatted }}</td></tr>{{ end }}</tbody>
    </table>
    <div class="summary"><div class="total">合计: {{ .Document.TotalAmountFormatted }}</div></div>
    <div class="signature"><div class="sign-box"><div class="sign-line"></div>制单</div><div class="sign-box"><div class="sign-line"></div>审核</div><div class="sign-box"><div class="sign-line"></div>确认</div></div>
</body>
</html>', 'A5', 'PORTRAIT', 5, 5, 5, 5, false, 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
ON CONFLICT (id) DO NOTHING;
