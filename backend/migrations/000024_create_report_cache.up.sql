-- Report Cache Tables for Pre-computed Report Data
-- These tables store aggregated report data to improve query performance

-- Report cache metadata table
CREATE TABLE IF NOT EXISTS report_cache_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    report_type VARCHAR(50) NOT NULL, -- 'SALES_SUMMARY', 'SALES_DAILY_TREND', 'INVENTORY_SUMMARY', etc.
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_report_cache_metadata UNIQUE (tenant_id, report_type, period_start, period_end)
);

CREATE INDEX idx_report_cache_metadata_tenant ON report_cache_metadata(tenant_id);
CREATE INDEX idx_report_cache_metadata_type ON report_cache_metadata(report_type);
CREATE INDEX idx_report_cache_metadata_valid ON report_cache_metadata(is_valid);
CREATE INDEX idx_report_cache_metadata_computed_at ON report_cache_metadata(computed_at);

-- Sales Summary cache
CREATE TABLE IF NOT EXISTS report_sales_summary_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    total_orders BIGINT NOT NULL DEFAULT 0,
    total_quantity DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_sales_amount DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_cost_amount DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_gross_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
    avg_order_value DECIMAL(20,4) NOT NULL DEFAULT 0,
    profit_margin DECIMAL(10,4) NOT NULL DEFAULT 0,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_sales_summary_cache UNIQUE (tenant_id, period_start, period_end)
);

CREATE INDEX idx_sales_summary_cache_tenant ON report_sales_summary_cache(tenant_id);
CREATE INDEX idx_sales_summary_cache_period ON report_sales_summary_cache(period_start, period_end);

-- Daily Sales Trend cache
CREATE TABLE IF NOT EXISTS report_sales_daily_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    date DATE NOT NULL,
    order_count BIGINT NOT NULL DEFAULT 0,
    total_amount DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
    items_sold DECIMAL(20,4) NOT NULL DEFAULT 0,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_sales_daily_cache UNIQUE (tenant_id, date)
);

CREATE INDEX idx_sales_daily_cache_tenant ON report_sales_daily_cache(tenant_id);
CREATE INDEX idx_sales_daily_cache_date ON report_sales_daily_cache(date);

-- Inventory Summary cache
CREATE TABLE IF NOT EXISTS report_inventory_summary_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    snapshot_date DATE NOT NULL,
    total_products BIGINT NOT NULL DEFAULT 0,
    total_quantity DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_value DECIMAL(20,4) NOT NULL DEFAULT 0,
    avg_turnover_rate DECIMAL(10,4) NOT NULL DEFAULT 0,
    low_stock_count BIGINT NOT NULL DEFAULT 0,
    out_of_stock_count BIGINT NOT NULL DEFAULT 0,
    overstock_count BIGINT NOT NULL DEFAULT 0,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_inventory_summary_cache UNIQUE (tenant_id, snapshot_date)
);

CREATE INDEX idx_inventory_summary_cache_tenant ON report_inventory_summary_cache(tenant_id);
CREATE INDEX idx_inventory_summary_cache_date ON report_inventory_summary_cache(snapshot_date);

-- Profit & Loss Summary cache (monthly)
CREATE TABLE IF NOT EXISTS report_pnl_monthly_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    year INT NOT NULL,
    month INT NOT NULL,
    sales_revenue DECIMAL(20,4) NOT NULL DEFAULT 0,
    sales_returns DECIMAL(20,4) NOT NULL DEFAULT 0,
    net_sales_revenue DECIMAL(20,4) NOT NULL DEFAULT 0,
    cogs DECIMAL(20,4) NOT NULL DEFAULT 0,
    gross_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
    gross_margin DECIMAL(10,4) NOT NULL DEFAULT 0,
    other_income DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_income DECIMAL(20,4) NOT NULL DEFAULT 0,
    expenses DECIMAL(20,4) NOT NULL DEFAULT 0,
    net_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
    net_margin DECIMAL(10,4) NOT NULL DEFAULT 0,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_pnl_monthly_cache UNIQUE (tenant_id, year, month)
);

CREATE INDEX idx_pnl_monthly_cache_tenant ON report_pnl_monthly_cache(tenant_id);
CREATE INDEX idx_pnl_monthly_cache_period ON report_pnl_monthly_cache(year, month);

-- Product Sales Ranking cache
CREATE TABLE IF NOT EXISTS report_product_ranking_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    rank INT NOT NULL,
    product_id UUID NOT NULL,
    product_sku VARCHAR(100) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    category_name VARCHAR(255),
    total_quantity DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_amount DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
    order_count BIGINT NOT NULL DEFAULT 0,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_product_ranking_cache UNIQUE (tenant_id, period_start, period_end, rank)
);

CREATE INDEX idx_product_ranking_cache_tenant ON report_product_ranking_cache(tenant_id);
CREATE INDEX idx_product_ranking_cache_period ON report_product_ranking_cache(period_start, period_end);

-- Customer Sales Ranking cache
CREATE TABLE IF NOT EXISTS report_customer_ranking_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    rank INT NOT NULL,
    customer_id UUID NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    total_orders BIGINT NOT NULL DEFAULT 0,
    total_quantity DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_amount DECIMAL(20,4) NOT NULL DEFAULT 0,
    total_profit DECIMAL(20,4) NOT NULL DEFAULT 0,
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_customer_ranking_cache UNIQUE (tenant_id, period_start, period_end, rank)
);

CREATE INDEX idx_customer_ranking_cache_tenant ON report_customer_ranking_cache(tenant_id);
CREATE INDEX idx_customer_ranking_cache_period ON report_customer_ranking_cache(period_start, period_end);

-- Report Scheduler Jobs table
CREATE TABLE IF NOT EXISTS report_scheduler_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- NULL means all tenants
    report_type VARCHAR(50) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL DEFAULT '0 2 * * *', -- Default 2am daily
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMP,
    last_run_status VARCHAR(20), -- 'SUCCESS', 'FAILED', 'RUNNING'
    last_error TEXT,
    next_run_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scheduler_jobs_tenant ON report_scheduler_jobs(tenant_id);
CREATE INDEX idx_scheduler_jobs_enabled ON report_scheduler_jobs(is_enabled);
CREATE INDEX idx_scheduler_jobs_next_run ON report_scheduler_jobs(next_run_at);

-- Add comment for documentation
COMMENT ON TABLE report_cache_metadata IS 'Tracks metadata for all cached report data';
COMMENT ON TABLE report_sales_summary_cache IS 'Pre-computed sales summary for performance';
COMMENT ON TABLE report_sales_daily_cache IS 'Pre-computed daily sales trend data';
COMMENT ON TABLE report_inventory_summary_cache IS 'Pre-computed inventory summary snapshots';
COMMENT ON TABLE report_pnl_monthly_cache IS 'Pre-computed monthly profit & loss data';
COMMENT ON TABLE report_product_ranking_cache IS 'Pre-computed product sales rankings';
COMMENT ON TABLE report_customer_ranking_cache IS 'Pre-computed customer sales rankings';
COMMENT ON TABLE report_scheduler_jobs IS 'Configuration for scheduled report generation';
