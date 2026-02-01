-- Migration: create_auto_print_rules
-- Created: 2026-02-01
-- Description: Create auto_print_rules table for automatic printing configuration

-- Create auto_print_rules table
CREATE TABLE IF NOT EXISTS auto_print_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    trigger_event VARCHAR(50) NOT NULL,
    template_id UUID,  -- Optional: specific template to use (NULL = use default)
    auto_print BOOLEAN NOT NULL DEFAULT FALSE,
    copies INTEGER NOT NULL DEFAULT 1,
    printer_name VARCHAR(100),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT chk_auto_print_rule_document_type CHECK (document_type IN (
        'SALES_ORDER', 'SALES_DELIVERY', 'SALES_RECEIPT', 'SALES_RETURN',
        'PURCHASE_ORDER', 'PURCHASE_RECEIVING', 'PURCHASE_RETURN',
        'RECEIPT_VOUCHER', 'PAYMENT_VOUCHER', 'STOCK_TAKING'
    )),
    CONSTRAINT chk_auto_print_rule_trigger_event CHECK (trigger_event IN (
        'CREATED', 'CONFIRMED', 'SHIPPED', 'RECEIVED', 'COMPLETED', 'PAID'
    )),
    CONSTRAINT chk_auto_print_rule_copies CHECK (copies >= 1 AND copies <= 100),

    -- Unique constraint: only one rule per (tenant, document_type, trigger_event) combination
    CONSTRAINT uq_auto_print_rule_tenant_doc_event UNIQUE (tenant_id, document_type, trigger_event)
);

-- Indexes for auto_print_rules
-- Index for tenant-based queries
CREATE INDEX IF NOT EXISTS idx_auto_print_rules_tenant ON auto_print_rules(tenant_id);

-- Composite index for the most common lookup pattern: finding enabled rules by doc type and event
CREATE INDEX IF NOT EXISTS idx_auto_print_rules_lookup ON auto_print_rules(tenant_id, document_type, trigger_event, enabled);

-- Index for finding all enabled rules for a tenant
CREATE INDEX IF NOT EXISTS idx_auto_print_rules_enabled ON auto_print_rules(tenant_id, enabled) WHERE enabled = TRUE;

-- Trigger for updated_at
CREATE TRIGGER update_auto_print_rules_updated_at
    BEFORE UPDATE ON auto_print_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE auto_print_rules IS 'Stores automatic printing rules that trigger print jobs when business events occur';
COMMENT ON COLUMN auto_print_rules.document_type IS 'Type of document this rule applies to (SALES_ORDER, PURCHASE_ORDER, etc.)';
COMMENT ON COLUMN auto_print_rules.trigger_event IS 'Business event that triggers printing (CREATED, CONFIRMED, SHIPPED, etc.)';
COMMENT ON COLUMN auto_print_rules.template_id IS 'Optional specific template to use; NULL means use the default template for the document type';
COMMENT ON COLUMN auto_print_rules.auto_print IS 'If TRUE, automatically send to printer; if FALSE, only generate PDF';
COMMENT ON COLUMN auto_print_rules.copies IS 'Number of copies to print (1-100)';
COMMENT ON COLUMN auto_print_rules.printer_name IS 'Optional specific printer name to use';
COMMENT ON COLUMN auto_print_rules.enabled IS 'Whether this rule is currently active';
