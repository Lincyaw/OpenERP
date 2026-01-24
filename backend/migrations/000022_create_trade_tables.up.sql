-- Create sales_orders table
CREATE TABLE IF NOT EXISTS sales_orders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    order_number VARCHAR(50) NOT NULL,
    customer_id UUID NOT NULL,
    customer_name VARCHAR(200) NOT NULL,
    warehouse_id UUID REFERENCES warehouses(id),
    total_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    payable_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    remark TEXT,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    shipped_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancel_reason VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sales_order_tenant_number UNIQUE (tenant_id, order_number)
);

CREATE INDEX IF NOT EXISTS idx_sales_orders_tenant_id ON sales_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_customer_id ON sales_orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_warehouse_id ON sales_orders(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_status ON sales_orders(status);
CREATE INDEX IF NOT EXISTS idx_sales_orders_confirmed_at ON sales_orders(confirmed_at);
CREATE INDEX IF NOT EXISTS idx_sales_orders_shipped_at ON sales_orders(shipped_at);

-- Create sales_order_items table
CREATE TABLE IF NOT EXISTS sales_order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES sales_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) NOT NULL,
    quantity DECIMAL(18,4) NOT NULL,
    unit_price DECIMAL(18,4) NOT NULL,
    amount DECIMAL(18,4) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    conversion_rate DECIMAL(18,6) NOT NULL DEFAULT 1,
    base_quantity DECIMAL(18,4) NOT NULL,
    base_unit VARCHAR(20) NOT NULL,
    remark VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_order_items_order_id ON sales_order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_sales_order_items_product_id ON sales_order_items(product_id);

-- Create purchase_orders table
CREATE TABLE IF NOT EXISTS purchase_orders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    order_number VARCHAR(50) NOT NULL,
    supplier_id UUID NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,
    warehouse_id UUID REFERENCES warehouses(id),
    total_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    payable_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    remark TEXT,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancel_reason VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_purchase_order_tenant_number UNIQUE (tenant_id, order_number)
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_tenant_id ON purchase_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier_id ON purchase_orders(supplier_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_warehouse_id ON purchase_orders(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_status ON purchase_orders(status);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_confirmed_at ON purchase_orders(confirmed_at);

-- Create purchase_order_items table
CREATE TABLE IF NOT EXISTS purchase_order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) NOT NULL,
    ordered_quantity DECIMAL(18,4) NOT NULL,
    received_quantity DECIMAL(18,4) NOT NULL DEFAULT 0,
    unit_cost DECIMAL(18,4) NOT NULL,
    amount DECIMAL(18,4) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    conversion_rate DECIMAL(18,6) NOT NULL DEFAULT 1,
    base_quantity DECIMAL(18,4) NOT NULL,
    base_unit VARCHAR(20) NOT NULL,
    remark VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_items_order_id ON purchase_order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_items_product_id ON purchase_order_items(product_id);

-- Create sales_returns table
CREATE TABLE IF NOT EXISTS sales_returns (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    return_number VARCHAR(50) NOT NULL,
    sales_order_id UUID NOT NULL REFERENCES sales_orders(id),
    sales_order_number VARCHAR(50) NOT NULL,
    customer_id UUID NOT NULL,
    customer_name VARCHAR(200) NOT NULL,
    warehouse_id UUID REFERENCES warehouses(id),
    total_refund DECIMAL(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    reason TEXT,
    remark TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by UUID,
    approval_note VARCHAR(500),
    rejected_at TIMESTAMP WITH TIME ZONE,
    rejected_by UUID,
    rejection_reason VARCHAR(500),
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancel_reason VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sales_return_tenant_number UNIQUE (tenant_id, return_number)
);

CREATE INDEX IF NOT EXISTS idx_sales_returns_tenant_id ON sales_returns(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sales_returns_sales_order_id ON sales_returns(sales_order_id);
CREATE INDEX IF NOT EXISTS idx_sales_returns_customer_id ON sales_returns(customer_id);
CREATE INDEX IF NOT EXISTS idx_sales_returns_warehouse_id ON sales_returns(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_sales_returns_status ON sales_returns(status);
CREATE INDEX IF NOT EXISTS idx_sales_returns_submitted_at ON sales_returns(submitted_at);
CREATE INDEX IF NOT EXISTS idx_sales_returns_approved_at ON sales_returns(approved_at);

-- Create sales_return_items table
CREATE TABLE IF NOT EXISTS sales_return_items (
    id UUID PRIMARY KEY,
    return_id UUID NOT NULL REFERENCES sales_returns(id) ON DELETE CASCADE,
    sales_order_item_id UUID NOT NULL,
    product_id UUID NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) NOT NULL,
    original_quantity DECIMAL(18,4) NOT NULL,
    return_quantity DECIMAL(18,4) NOT NULL,
    unit_price DECIMAL(18,4) NOT NULL,
    refund_amount DECIMAL(18,4) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    reason VARCHAR(500),
    condition_on_return VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_return_items_return_id ON sales_return_items(return_id);
CREATE INDEX IF NOT EXISTS idx_sales_return_items_product_id ON sales_return_items(product_id);
CREATE INDEX IF NOT EXISTS idx_sales_return_items_sales_order_item_id ON sales_return_items(sales_order_item_id);

-- Create purchase_returns table
CREATE TABLE IF NOT EXISTS purchase_returns (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    return_number VARCHAR(50) NOT NULL,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id),
    purchase_order_number VARCHAR(50) NOT NULL,
    supplier_id UUID NOT NULL,
    supplier_name VARCHAR(200) NOT NULL,
    warehouse_id UUID REFERENCES warehouses(id),
    total_refund DECIMAL(18,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    reason TEXT,
    remark TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by UUID,
    approval_note VARCHAR(500),
    rejected_at TIMESTAMP WITH TIME ZONE,
    rejected_by UUID,
    rejection_reason VARCHAR(500),
    shipped_at TIMESTAMP WITH TIME ZONE,
    shipped_by UUID,
    shipping_note VARCHAR(500),
    tracking_number VARCHAR(100),
    completed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancel_reason VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_purchase_return_tenant_number UNIQUE (tenant_id, return_number)
);

CREATE INDEX IF NOT EXISTS idx_purchase_returns_tenant_id ON purchase_returns(tenant_id);
CREATE INDEX IF NOT EXISTS idx_purchase_returns_purchase_order_id ON purchase_returns(purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_returns_supplier_id ON purchase_returns(supplier_id);
CREATE INDEX IF NOT EXISTS idx_purchase_returns_warehouse_id ON purchase_returns(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_purchase_returns_status ON purchase_returns(status);
CREATE INDEX IF NOT EXISTS idx_purchase_returns_submitted_at ON purchase_returns(submitted_at);
CREATE INDEX IF NOT EXISTS idx_purchase_returns_shipped_at ON purchase_returns(shipped_at);

-- Create purchase_return_items table
CREATE TABLE IF NOT EXISTS purchase_return_items (
    id UUID PRIMARY KEY,
    return_id UUID NOT NULL REFERENCES purchase_returns(id) ON DELETE CASCADE,
    purchase_order_item_id UUID NOT NULL,
    product_id UUID NOT NULL,
    product_name VARCHAR(200) NOT NULL,
    product_code VARCHAR(50) NOT NULL,
    original_quantity DECIMAL(18,4) NOT NULL,
    return_quantity DECIMAL(18,4) NOT NULL,
    unit_cost DECIMAL(18,4) NOT NULL,
    refund_amount DECIMAL(18,4) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    reason VARCHAR(500),
    condition_on_return VARCHAR(100),
    batch_number VARCHAR(50),
    shipped_quantity DECIMAL(18,4) NOT NULL DEFAULT 0,
    shipped_at TIMESTAMP WITH TIME ZONE,
    supplier_received_qty DECIMAL(18,4) NOT NULL DEFAULT 0,
    supplier_received_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_return_items_return_id ON purchase_return_items(return_id);
CREATE INDEX IF NOT EXISTS idx_purchase_return_items_product_id ON purchase_return_items(product_id);
CREATE INDEX IF NOT EXISTS idx_purchase_return_items_purchase_order_item_id ON purchase_return_items(purchase_order_item_id);

-- Add comments
COMMENT ON TABLE sales_orders IS 'Sales orders for customer purchases';
COMMENT ON TABLE sales_order_items IS 'Line items for sales orders';
COMMENT ON TABLE purchase_orders IS 'Purchase orders for supplier procurement';
COMMENT ON TABLE purchase_order_items IS 'Line items for purchase orders';
COMMENT ON TABLE sales_returns IS 'Returns from customers for sales orders';
COMMENT ON TABLE sales_return_items IS 'Line items for sales returns';
COMMENT ON TABLE purchase_returns IS 'Returns to suppliers for purchase orders';
COMMENT ON TABLE purchase_return_items IS 'Line items for purchase returns';
