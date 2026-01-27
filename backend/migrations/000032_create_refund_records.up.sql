-- Create refund_records table for tracking payment gateway refunds
CREATE TABLE IF NOT EXISTS refund_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    created_by UUID,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Refund identification
    refund_number VARCHAR(50) NOT NULL,

    -- Original payment reference
    original_payment_id UUID,
    original_order_id UUID NOT NULL,
    original_order_number VARCHAR(50) NOT NULL,

    -- Source document reference
    source_type VARCHAR(30) NOT NULL,
    source_id UUID NOT NULL,
    source_number VARCHAR(50),

    -- Customer information
    customer_id UUID NOT NULL,
    customer_name VARCHAR(200) NOT NULL,

    -- Refund amounts
    refund_amount DECIMAL(18,4) NOT NULL,
    actual_refund_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'CNY',

    -- Gateway information
    gateway_type VARCHAR(20) NOT NULL,
    gateway_refund_id VARCHAR(100),
    gateway_order_id VARCHAR(100),
    gateway_transaction_id VARCHAR(100),

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    reason VARCHAR(500),
    remark TEXT,
    fail_reason VARCHAR(500),
    raw_response TEXT,

    -- Timestamps
    requested_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT refund_records_tenant_fk FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT refund_records_customer_fk FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT,
    CONSTRAINT refund_records_positive_amount CHECK (refund_amount > 0),
    CONSTRAINT refund_records_actual_amount_non_negative CHECK (actual_refund_amount >= 0),
    CONSTRAINT refund_records_valid_status CHECK (status IN ('PENDING', 'PROCESSING', 'SUCCESS', 'FAILED', 'CLOSED')),
    CONSTRAINT refund_records_valid_source_type CHECK (source_type IN ('SALES_RETURN', 'CREDIT_MEMO', 'ORDER_CANCEL', 'MANUAL')),
    CONSTRAINT refund_records_valid_gateway_type CHECK (gateway_type IN ('WECHAT', 'ALIPAY'))
);

-- Create indexes for common queries
CREATE UNIQUE INDEX idx_refund_tenant_number ON refund_records(tenant_id, refund_number);
CREATE INDEX idx_refund_tenant_id ON refund_records(tenant_id);
CREATE INDEX idx_refund_customer_id ON refund_records(customer_id);
CREATE INDEX idx_refund_source ON refund_records(tenant_id, source_type, source_id);
CREATE INDEX idx_refund_gateway_type ON refund_records(gateway_type);
CREATE INDEX idx_refund_gateway_refund_id ON refund_records(gateway_type, gateway_refund_id);
CREATE INDEX idx_refund_status ON refund_records(tenant_id, status);
CREATE INDEX idx_refund_original_order ON refund_records(tenant_id, original_order_id);
CREATE INDEX idx_refund_requested_at ON refund_records(tenant_id, requested_at);
CREATE INDEX idx_refund_completed_at ON refund_records(tenant_id, completed_at);

-- Add comment for documentation
COMMENT ON TABLE refund_records IS 'Tracks payment gateway refunds (WeChat/Alipay) and their status';
COMMENT ON COLUMN refund_records.refund_number IS 'Internal refund reference number';
COMMENT ON COLUMN refund_records.source_type IS 'Type of document that triggered the refund: SALES_RETURN, CREDIT_MEMO, ORDER_CANCEL, MANUAL';
COMMENT ON COLUMN refund_records.gateway_type IS 'Payment gateway type: WECHAT, ALIPAY';
COMMENT ON COLUMN refund_records.gateway_refund_id IS 'Refund ID returned by the payment gateway';
COMMENT ON COLUMN refund_records.status IS 'Refund status: PENDING, PROCESSING, SUCCESS, FAILED, CLOSED';
