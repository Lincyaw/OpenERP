-- Migration: Create usage metering tables
-- Description: Implements usage records and quotas for SaaS billing

-- Create usage_records table for tracking all usage events
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    usage_type VARCHAR(50) NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity >= 0),
    unit VARCHAR(20) NOT NULL DEFAULT 'requests',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    source_type VARCHAR(100),
    source_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure period_end is after period_start
    CONSTRAINT chk_usage_records_period CHECK (period_end > period_start),

    -- Validate usage_type enum values
    CONSTRAINT chk_usage_records_type CHECK (usage_type IN (
        'API_CALLS', 'STORAGE_BYTES', 'ACTIVE_USERS', 'ORDERS_CREATED',
        'PRODUCTS_SKU', 'WAREHOUSES', 'CUSTOMERS', 'SUPPLIERS',
        'REPORTS_GENERATED', 'DATA_EXPORTS', 'DATA_IMPORT_ROWS',
        'INTEGRATION_CALLS', 'NOTIFICATIONS_SENT', 'ATTACHMENT_BYTES'
    )),

    -- Validate unit enum values
    CONSTRAINT chk_usage_records_unit CHECK (unit IN ('requests', 'bytes', 'count'))
);

-- Create indexes for usage_records
CREATE INDEX idx_usage_records_tenant_id ON usage_records(tenant_id);
CREATE INDEX idx_usage_records_usage_type ON usage_records(usage_type);
CREATE INDEX idx_usage_records_recorded_at ON usage_records(recorded_at);
CREATE INDEX idx_usage_records_period ON usage_records(period_start, period_end);
CREATE INDEX idx_usage_records_tenant_type_period ON usage_records(tenant_id, usage_type, period_start, period_end);
CREATE INDEX idx_usage_records_source ON usage_records(source_type, source_id);
CREATE INDEX idx_usage_records_user_id ON usage_records(user_id) WHERE user_id IS NOT NULL;

-- Create partial index for recent records (last 90 days) for faster queries
CREATE INDEX idx_usage_records_recent ON usage_records(tenant_id, usage_type, recorded_at)
    WHERE recorded_at > NOW() - INTERVAL '90 days';

-- Add update trigger for updated_at
CREATE TRIGGER trg_usage_records_updated_at
    BEFORE UPDATE ON usage_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for usage_records
COMMENT ON TABLE usage_records IS 'Immutable records of usage events for billing and metering';
COMMENT ON COLUMN usage_records.tenant_id IS 'The tenant this usage belongs to';
COMMENT ON COLUMN usage_records.usage_type IS 'Type of usage being recorded (API_CALLS, STORAGE_BYTES, etc.)';
COMMENT ON COLUMN usage_records.quantity IS 'Amount of usage (always positive)';
COMMENT ON COLUMN usage_records.unit IS 'Unit of measurement (requests, bytes, count)';
COMMENT ON COLUMN usage_records.recorded_at IS 'When the usage event occurred';
COMMENT ON COLUMN usage_records.period_start IS 'Start of the billing period';
COMMENT ON COLUMN usage_records.period_end IS 'End of the billing period';
COMMENT ON COLUMN usage_records.source_type IS 'Source of the usage event (e.g., sales_order, api_request)';
COMMENT ON COLUMN usage_records.source_id IS 'ID of the source entity';
COMMENT ON COLUMN usage_records.metadata IS 'Additional context about the usage';
COMMENT ON COLUMN usage_records.user_id IS 'User who triggered the usage (optional)';
COMMENT ON COLUMN usage_records.ip_address IS 'IP address of the request (for API calls)';
COMMENT ON COLUMN usage_records.user_agent IS 'User agent of the request (for API calls)';

-- Create usage_quotas table for defining usage limits
CREATE TABLE usage_quotas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_id VARCHAR(50) NOT NULL CHECK (plan_id IN ('free', 'basic', 'pro', 'enterprise')),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    usage_type VARCHAR(50) NOT NULL,
    quota_limit BIGINT NOT NULL DEFAULT -1,
    unit VARCHAR(20) NOT NULL DEFAULT 'requests',
    reset_period VARCHAR(20) NOT NULL DEFAULT 'MONTHLY',
    soft_limit BIGINT,
    overage_policy VARCHAR(20) NOT NULL DEFAULT 'BLOCK',
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Validate usage_type enum values
    CONSTRAINT chk_usage_quotas_type CHECK (usage_type IN (
        'API_CALLS', 'STORAGE_BYTES', 'ACTIVE_USERS', 'ORDERS_CREATED',
        'PRODUCTS_SKU', 'WAREHOUSES', 'CUSTOMERS', 'SUPPLIERS',
        'REPORTS_GENERATED', 'DATA_EXPORTS', 'DATA_IMPORT_ROWS',
        'INTEGRATION_CALLS', 'NOTIFICATIONS_SENT', 'ATTACHMENT_BYTES'
    )),

    -- Validate unit enum values
    CONSTRAINT chk_usage_quotas_unit CHECK (unit IN ('requests', 'bytes', 'count')),

    -- Validate reset_period enum values
    CONSTRAINT chk_usage_quotas_reset_period CHECK (reset_period IN (
        'DAILY', 'WEEKLY', 'MONTHLY', 'YEARLY', 'NEVER'
    )),

    -- Validate overage_policy enum values
    CONSTRAINT chk_usage_quotas_overage_policy CHECK (overage_policy IN (
        'BLOCK', 'WARN', 'CHARGE', 'THROTTLE'
    )),

    -- Validate limit is -1 (unlimited) or non-negative
    CONSTRAINT chk_usage_quotas_limit CHECK (quota_limit >= -1),

    -- Validate soft_limit is less than hard limit when both are set
    CONSTRAINT chk_usage_quotas_soft_limit CHECK (
        soft_limit IS NULL OR
        quota_limit = -1 OR
        soft_limit < quota_limit
    ),

    -- Ensure unique quota per plan+type (for plan defaults) or tenant+type (for overrides)
    CONSTRAINT uq_usage_quotas_plan_type UNIQUE (plan_id, usage_type) WHERE tenant_id IS NULL,
    CONSTRAINT uq_usage_quotas_tenant_type UNIQUE (tenant_id, usage_type) WHERE tenant_id IS NOT NULL
);

-- Create indexes for usage_quotas
CREATE INDEX idx_usage_quotas_plan_id ON usage_quotas(plan_id);
CREATE INDEX idx_usage_quotas_tenant_id ON usage_quotas(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_usage_quotas_usage_type ON usage_quotas(usage_type);
CREATE INDEX idx_usage_quotas_is_active ON usage_quotas(is_active);
CREATE INDEX idx_usage_quotas_plan_type_active ON usage_quotas(plan_id, usage_type) WHERE is_active = true;

-- Add update trigger for updated_at
CREATE TRIGGER trg_usage_quotas_updated_at
    BEFORE UPDATE ON usage_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for usage_quotas
COMMENT ON TABLE usage_quotas IS 'Usage quota definitions for subscription plans and tenant overrides';
COMMENT ON COLUMN usage_quotas.plan_id IS 'Subscription plan ID (free, basic, pro, enterprise)';
COMMENT ON COLUMN usage_quotas.tenant_id IS 'Optional tenant-specific override (NULL = plan default)';
COMMENT ON COLUMN usage_quotas.usage_type IS 'Type of usage being limited';
COMMENT ON COLUMN usage_quotas.quota_limit IS 'Maximum allowed usage (-1 = unlimited)';
COMMENT ON COLUMN usage_quotas.unit IS 'Unit of measurement';
COMMENT ON COLUMN usage_quotas.reset_period IS 'When the quota resets (DAILY, WEEKLY, MONTHLY, YEARLY, NEVER)';
COMMENT ON COLUMN usage_quotas.soft_limit IS 'Optional soft limit for warnings';
COMMENT ON COLUMN usage_quotas.overage_policy IS 'What happens when quota is exceeded (BLOCK, WARN, CHARGE, THROTTLE)';
COMMENT ON COLUMN usage_quotas.description IS 'Human-readable description';
COMMENT ON COLUMN usage_quotas.is_active IS 'Whether this quota is currently active';

-- Insert default quotas for FREE plan
INSERT INTO usage_quotas (plan_id, usage_type, quota_limit, unit, reset_period, soft_limit, overage_policy, description) VALUES
    ('free', 'API_CALLS', 10000, 'requests', 'MONTHLY', 8000, 'BLOCK', 'API calls per month'),
    ('free', 'STORAGE_BYTES', 104857600, 'bytes', 'NEVER', 83886080, 'BLOCK', 'Total storage (100 MB)'),
    ('free', 'ACTIVE_USERS', 3, 'count', 'NEVER', NULL, 'BLOCK', 'Maximum active users'),
    ('free', 'ORDERS_CREATED', 100, 'count', 'MONTHLY', 80, 'BLOCK', 'Orders created per month'),
    ('free', 'PRODUCTS_SKU', 100, 'count', 'NEVER', 80, 'BLOCK', 'Maximum products/SKUs'),
    ('free', 'WAREHOUSES', 1, 'count', 'NEVER', NULL, 'BLOCK', 'Maximum warehouses'),
    ('free', 'CUSTOMERS', 50, 'count', 'NEVER', 40, 'BLOCK', 'Maximum customers'),
    ('free', 'SUPPLIERS', 20, 'count', 'NEVER', 15, 'BLOCK', 'Maximum suppliers'),
    ('free', 'REPORTS_GENERATED', 10, 'requests', 'MONTHLY', 8, 'BLOCK', 'Reports generated per month'),
    ('free', 'DATA_EXPORTS', 5, 'requests', 'MONTHLY', NULL, 'BLOCK', 'Data exports per month'),
    ('free', 'DATA_IMPORT_ROWS', 100, 'count', 'MONTHLY', NULL, 'BLOCK', 'Import rows per month'),
    ('free', 'INTEGRATION_CALLS', 0, 'requests', 'MONTHLY', NULL, 'BLOCK', 'Integration API calls (disabled)'),
    ('free', 'NOTIFICATIONS_SENT', 0, 'requests', 'MONTHLY', NULL, 'BLOCK', 'Notifications (disabled)'),
    ('free', 'ATTACHMENT_BYTES', 52428800, 'bytes', 'NEVER', 41943040, 'BLOCK', 'Attachment storage (50 MB)');

-- Insert default quotas for BASIC plan
INSERT INTO usage_quotas (plan_id, usage_type, quota_limit, unit, reset_period, soft_limit, overage_policy, description) VALUES
    ('basic', 'API_CALLS', 100000, 'requests', 'MONTHLY', 80000, 'WARN', 'API calls per month'),
    ('basic', 'STORAGE_BYTES', 1073741824, 'bytes', 'NEVER', 858993459, 'WARN', 'Total storage (1 GB)'),
    ('basic', 'ACTIVE_USERS', 10, 'count', 'NEVER', 8, 'BLOCK', 'Maximum active users'),
    ('basic', 'ORDERS_CREATED', 1000, 'count', 'MONTHLY', 800, 'WARN', 'Orders created per month'),
    ('basic', 'PRODUCTS_SKU', 1000, 'count', 'NEVER', 800, 'WARN', 'Maximum products/SKUs'),
    ('basic', 'WAREHOUSES', 3, 'count', 'NEVER', NULL, 'BLOCK', 'Maximum warehouses'),
    ('basic', 'CUSTOMERS', 500, 'count', 'NEVER', 400, 'WARN', 'Maximum customers'),
    ('basic', 'SUPPLIERS', 100, 'count', 'NEVER', 80, 'WARN', 'Maximum suppliers'),
    ('basic', 'REPORTS_GENERATED', 100, 'requests', 'MONTHLY', 80, 'WARN', 'Reports generated per month'),
    ('basic', 'DATA_EXPORTS', 50, 'requests', 'MONTHLY', 40, 'WARN', 'Data exports per month'),
    ('basic', 'DATA_IMPORT_ROWS', 1000, 'count', 'MONTHLY', 800, 'WARN', 'Import rows per month'),
    ('basic', 'INTEGRATION_CALLS', 0, 'requests', 'MONTHLY', NULL, 'BLOCK', 'Integration API calls (disabled)'),
    ('basic', 'NOTIFICATIONS_SENT', 500, 'requests', 'MONTHLY', 400, 'WARN', 'Notifications per month'),
    ('basic', 'ATTACHMENT_BYTES', 536870912, 'bytes', 'NEVER', 429496730, 'WARN', 'Attachment storage (500 MB)');

-- Insert default quotas for PRO plan
INSERT INTO usage_quotas (plan_id, usage_type, quota_limit, unit, reset_period, soft_limit, overage_policy, description) VALUES
    ('pro', 'API_CALLS', 1000000, 'requests', 'MONTHLY', 800000, 'CHARGE', 'API calls per month'),
    ('pro', 'STORAGE_BYTES', 10737418240, 'bytes', 'NEVER', 8589934592, 'CHARGE', 'Total storage (10 GB)'),
    ('pro', 'ACTIVE_USERS', 50, 'count', 'NEVER', 40, 'WARN', 'Maximum active users'),
    ('pro', 'ORDERS_CREATED', 10000, 'count', 'MONTHLY', 8000, 'CHARGE', 'Orders created per month'),
    ('pro', 'PRODUCTS_SKU', 10000, 'count', 'NEVER', 8000, 'WARN', 'Maximum products/SKUs'),
    ('pro', 'WAREHOUSES', 10, 'count', 'NEVER', 8, 'WARN', 'Maximum warehouses'),
    ('pro', 'CUSTOMERS', 5000, 'count', 'NEVER', 4000, 'WARN', 'Maximum customers'),
    ('pro', 'SUPPLIERS', 1000, 'count', 'NEVER', 800, 'WARN', 'Maximum suppliers'),
    ('pro', 'REPORTS_GENERATED', 1000, 'requests', 'MONTHLY', 800, 'CHARGE', 'Reports generated per month'),
    ('pro', 'DATA_EXPORTS', 500, 'requests', 'MONTHLY', 400, 'CHARGE', 'Data exports per month'),
    ('pro', 'DATA_IMPORT_ROWS', 10000, 'count', 'MONTHLY', 8000, 'CHARGE', 'Import rows per month'),
    ('pro', 'INTEGRATION_CALLS', 50000, 'requests', 'MONTHLY', 40000, 'CHARGE', 'Integration API calls per month'),
    ('pro', 'NOTIFICATIONS_SENT', 5000, 'requests', 'MONTHLY', 4000, 'CHARGE', 'Notifications per month'),
    ('pro', 'ATTACHMENT_BYTES', 5368709120, 'bytes', 'NEVER', 4294967296, 'CHARGE', 'Attachment storage (5 GB)');

-- Insert default quotas for ENTERPRISE plan (unlimited for most)
INSERT INTO usage_quotas (plan_id, usage_type, quota_limit, unit, reset_period, soft_limit, overage_policy, description) VALUES
    ('enterprise', 'API_CALLS', -1, 'requests', 'MONTHLY', NULL, 'WARN', 'API calls (unlimited)'),
    ('enterprise', 'STORAGE_BYTES', -1, 'bytes', 'NEVER', NULL, 'WARN', 'Total storage (unlimited)'),
    ('enterprise', 'ACTIVE_USERS', -1, 'count', 'NEVER', NULL, 'WARN', 'Active users (unlimited)'),
    ('enterprise', 'ORDERS_CREATED', -1, 'count', 'MONTHLY', NULL, 'WARN', 'Orders created (unlimited)'),
    ('enterprise', 'PRODUCTS_SKU', -1, 'count', 'NEVER', NULL, 'WARN', 'Products/SKUs (unlimited)'),
    ('enterprise', 'WAREHOUSES', -1, 'count', 'NEVER', NULL, 'WARN', 'Warehouses (unlimited)'),
    ('enterprise', 'CUSTOMERS', -1, 'count', 'NEVER', NULL, 'WARN', 'Customers (unlimited)'),
    ('enterprise', 'SUPPLIERS', -1, 'count', 'NEVER', NULL, 'WARN', 'Suppliers (unlimited)'),
    ('enterprise', 'REPORTS_GENERATED', -1, 'requests', 'MONTHLY', NULL, 'WARN', 'Reports (unlimited)'),
    ('enterprise', 'DATA_EXPORTS', -1, 'requests', 'MONTHLY', NULL, 'WARN', 'Data exports (unlimited)'),
    ('enterprise', 'DATA_IMPORT_ROWS', -1, 'count', 'MONTHLY', NULL, 'WARN', 'Import rows (unlimited)'),
    ('enterprise', 'INTEGRATION_CALLS', -1, 'requests', 'MONTHLY', NULL, 'WARN', 'Integration calls (unlimited)'),
    ('enterprise', 'NOTIFICATIONS_SENT', -1, 'requests', 'MONTHLY', NULL, 'WARN', 'Notifications (unlimited)'),
    ('enterprise', 'ATTACHMENT_BYTES', -1, 'bytes', 'NEVER', NULL, 'WARN', 'Attachment storage (unlimited)');
