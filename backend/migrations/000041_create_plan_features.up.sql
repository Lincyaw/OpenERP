-- Migration: Create plan_features table
-- Description: Implements plan-feature mapping for SaaS subscription tiers

-- Create plan_features table
CREATE TABLE plan_features (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plan_id VARCHAR(50) NOT NULL CHECK (plan_id IN ('free', 'basic', 'pro', 'enterprise')),
    feature_key VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    feature_limit INTEGER,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique feature per plan
    CONSTRAINT uq_plan_feature UNIQUE (plan_id, feature_key)
);

-- Create indexes for plan_features
CREATE INDEX idx_plan_features_plan_id ON plan_features(plan_id);
CREATE INDEX idx_plan_features_feature_key ON plan_features(feature_key);
CREATE INDEX idx_plan_features_enabled ON plan_features(enabled);

-- Add update trigger for updated_at
CREATE TRIGGER trg_plan_features_updated_at
    BEFORE UPDATE ON plan_features
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for plan_features
COMMENT ON TABLE plan_features IS 'Feature mappings for subscription plans';
COMMENT ON COLUMN plan_features.plan_id IS 'Subscription plan identifier (free, basic, pro, enterprise)';
COMMENT ON COLUMN plan_features.feature_key IS 'Unique identifier for the feature';
COMMENT ON COLUMN plan_features.enabled IS 'Whether the feature is enabled for this plan';
COMMENT ON COLUMN plan_features.feature_limit IS 'Optional limit for the feature (NULL = unlimited)';
COMMENT ON COLUMN plan_features.description IS 'Human-readable description of the feature';

-- Insert default features for FREE plan
INSERT INTO plan_features (plan_id, feature_key, enabled, feature_limit, description) VALUES
    ('free', 'multi_warehouse', false, NULL, 'Multiple warehouse management'),
    ('free', 'batch_management', false, NULL, 'Batch/lot tracking'),
    ('free', 'serial_tracking', false, NULL, 'Serial number tracking'),
    ('free', 'multi_currency', false, NULL, 'Multi-currency support'),
    ('free', 'advanced_reporting', false, NULL, 'Advanced analytics and reports'),
    ('free', 'api_access', false, NULL, 'API access for integrations'),
    ('free', 'custom_fields', false, NULL, 'Custom fields on entities'),
    ('free', 'audit_log', false, NULL, 'Audit log tracking'),
    ('free', 'data_export', true, NULL, 'Export data to CSV/Excel'),
    ('free', 'data_import', true, 100, 'Import data from CSV (100 rows/import)'),
    ('free', 'sales_orders', true, NULL, 'Create and manage sales orders'),
    ('free', 'purchase_orders', true, NULL, 'Create and manage purchase orders'),
    ('free', 'sales_returns', true, NULL, 'Process sales returns'),
    ('free', 'purchase_returns', true, NULL, 'Process purchase returns'),
    ('free', 'quotations', false, NULL, 'Create quotations'),
    ('free', 'price_management', false, NULL, 'Advanced price management'),
    ('free', 'discount_rules', false, NULL, 'Discount rules engine'),
    ('free', 'credit_management', false, NULL, 'Customer credit management'),
    ('free', 'receivables', true, NULL, 'Accounts receivable tracking'),
    ('free', 'payables', true, NULL, 'Accounts payable tracking'),
    ('free', 'reconciliation', false, NULL, 'Account reconciliation'),
    ('free', 'expense_tracking', false, NULL, 'Expense tracking'),
    ('free', 'financial_reports', false, NULL, 'Financial reports'),
    ('free', 'workflow_approval', false, NULL, 'Workflow approval system'),
    ('free', 'notifications', false, NULL, 'Email/SMS notifications'),
    ('free', 'integrations', false, NULL, 'Third-party integrations'),
    ('free', 'white_labeling', false, NULL, 'White-label branding'),
    ('free', 'priority_support', false, NULL, 'Priority support'),
    ('free', 'dedicated_support', false, NULL, 'Dedicated support manager'),
    ('free', 'sla', false, NULL, 'Service level agreement');

-- Insert default features for BASIC plan
INSERT INTO plan_features (plan_id, feature_key, enabled, feature_limit, description) VALUES
    ('basic', 'multi_warehouse', true, NULL, 'Multiple warehouse management'),
    ('basic', 'batch_management', true, NULL, 'Batch/lot tracking'),
    ('basic', 'serial_tracking', false, NULL, 'Serial number tracking'),
    ('basic', 'multi_currency', false, NULL, 'Multi-currency support'),
    ('basic', 'advanced_reporting', false, NULL, 'Advanced analytics and reports'),
    ('basic', 'api_access', false, NULL, 'API access for integrations'),
    ('basic', 'custom_fields', false, NULL, 'Custom fields on entities'),
    ('basic', 'audit_log', true, NULL, 'Audit log tracking'),
    ('basic', 'data_export', true, NULL, 'Export data to CSV/Excel'),
    ('basic', 'data_import', true, 1000, 'Import data from CSV (1000 rows/import)'),
    ('basic', 'sales_orders', true, NULL, 'Create and manage sales orders'),
    ('basic', 'purchase_orders', true, NULL, 'Create and manage purchase orders'),
    ('basic', 'sales_returns', true, NULL, 'Process sales returns'),
    ('basic', 'purchase_returns', true, NULL, 'Process purchase returns'),
    ('basic', 'quotations', true, NULL, 'Create quotations'),
    ('basic', 'price_management', true, NULL, 'Advanced price management'),
    ('basic', 'discount_rules', false, NULL, 'Discount rules engine'),
    ('basic', 'credit_management', true, NULL, 'Customer credit management'),
    ('basic', 'receivables', true, NULL, 'Accounts receivable tracking'),
    ('basic', 'payables', true, NULL, 'Accounts payable tracking'),
    ('basic', 'reconciliation', true, NULL, 'Account reconciliation'),
    ('basic', 'expense_tracking', true, NULL, 'Expense tracking'),
    ('basic', 'financial_reports', false, NULL, 'Financial reports'),
    ('basic', 'workflow_approval', false, NULL, 'Workflow approval system'),
    ('basic', 'notifications', true, NULL, 'Email/SMS notifications'),
    ('basic', 'integrations', false, NULL, 'Third-party integrations'),
    ('basic', 'white_labeling', false, NULL, 'White-label branding'),
    ('basic', 'priority_support', false, NULL, 'Priority support'),
    ('basic', 'dedicated_support', false, NULL, 'Dedicated support manager'),
    ('basic', 'sla', false, NULL, 'Service level agreement');

-- Insert default features for PRO plan
INSERT INTO plan_features (plan_id, feature_key, enabled, feature_limit, description) VALUES
    ('pro', 'multi_warehouse', true, NULL, 'Multiple warehouse management'),
    ('pro', 'batch_management', true, NULL, 'Batch/lot tracking'),
    ('pro', 'serial_tracking', true, NULL, 'Serial number tracking'),
    ('pro', 'multi_currency', true, NULL, 'Multi-currency support'),
    ('pro', 'advanced_reporting', true, NULL, 'Advanced analytics and reports'),
    ('pro', 'api_access', true, NULL, 'API access for integrations'),
    ('pro', 'custom_fields', true, NULL, 'Custom fields on entities'),
    ('pro', 'audit_log', true, NULL, 'Audit log tracking'),
    ('pro', 'data_export', true, NULL, 'Export data to CSV/Excel'),
    ('pro', 'data_import', true, 10000, 'Import data from CSV (10000 rows/import)'),
    ('pro', 'sales_orders', true, NULL, 'Create and manage sales orders'),
    ('pro', 'purchase_orders', true, NULL, 'Create and manage purchase orders'),
    ('pro', 'sales_returns', true, NULL, 'Process sales returns'),
    ('pro', 'purchase_returns', true, NULL, 'Process purchase returns'),
    ('pro', 'quotations', true, NULL, 'Create quotations'),
    ('pro', 'price_management', true, NULL, 'Advanced price management'),
    ('pro', 'discount_rules', true, NULL, 'Discount rules engine'),
    ('pro', 'credit_management', true, NULL, 'Customer credit management'),
    ('pro', 'receivables', true, NULL, 'Accounts receivable tracking'),
    ('pro', 'payables', true, NULL, 'Accounts payable tracking'),
    ('pro', 'reconciliation', true, NULL, 'Account reconciliation'),
    ('pro', 'expense_tracking', true, NULL, 'Expense tracking'),
    ('pro', 'financial_reports', true, NULL, 'Financial reports'),
    ('pro', 'workflow_approval', true, NULL, 'Workflow approval system'),
    ('pro', 'notifications', true, NULL, 'Email/SMS notifications'),
    ('pro', 'integrations', true, NULL, 'Third-party integrations'),
    ('pro', 'white_labeling', false, NULL, 'White-label branding'),
    ('pro', 'priority_support', true, NULL, 'Priority support'),
    ('pro', 'dedicated_support', false, NULL, 'Dedicated support manager'),
    ('pro', 'sla', false, NULL, 'Service level agreement');

-- Insert default features for ENTERPRISE plan
INSERT INTO plan_features (plan_id, feature_key, enabled, feature_limit, description) VALUES
    ('enterprise', 'multi_warehouse', true, NULL, 'Multiple warehouse management'),
    ('enterprise', 'batch_management', true, NULL, 'Batch/lot tracking'),
    ('enterprise', 'serial_tracking', true, NULL, 'Serial number tracking'),
    ('enterprise', 'multi_currency', true, NULL, 'Multi-currency support'),
    ('enterprise', 'advanced_reporting', true, NULL, 'Advanced analytics and reports'),
    ('enterprise', 'api_access', true, NULL, 'API access for integrations'),
    ('enterprise', 'custom_fields', true, NULL, 'Custom fields on entities'),
    ('enterprise', 'audit_log', true, NULL, 'Audit log tracking'),
    ('enterprise', 'data_export', true, NULL, 'Export data to CSV/Excel'),
    ('enterprise', 'data_import', true, NULL, 'Import data from CSV (unlimited)'),
    ('enterprise', 'sales_orders', true, NULL, 'Create and manage sales orders'),
    ('enterprise', 'purchase_orders', true, NULL, 'Create and manage purchase orders'),
    ('enterprise', 'sales_returns', true, NULL, 'Process sales returns'),
    ('enterprise', 'purchase_returns', true, NULL, 'Process purchase returns'),
    ('enterprise', 'quotations', true, NULL, 'Create quotations'),
    ('enterprise', 'price_management', true, NULL, 'Advanced price management'),
    ('enterprise', 'discount_rules', true, NULL, 'Discount rules engine'),
    ('enterprise', 'credit_management', true, NULL, 'Customer credit management'),
    ('enterprise', 'receivables', true, NULL, 'Accounts receivable tracking'),
    ('enterprise', 'payables', true, NULL, 'Accounts payable tracking'),
    ('enterprise', 'reconciliation', true, NULL, 'Account reconciliation'),
    ('enterprise', 'expense_tracking', true, NULL, 'Expense tracking'),
    ('enterprise', 'financial_reports', true, NULL, 'Financial reports'),
    ('enterprise', 'workflow_approval', true, NULL, 'Workflow approval system'),
    ('enterprise', 'notifications', true, NULL, 'Email/SMS notifications'),
    ('enterprise', 'integrations', true, NULL, 'Third-party integrations'),
    ('enterprise', 'white_labeling', true, NULL, 'White-label branding'),
    ('enterprise', 'priority_support', true, NULL, 'Priority support'),
    ('enterprise', 'dedicated_support', true, NULL, 'Dedicated support manager'),
    ('enterprise', 'sla', true, NULL, 'Service level agreement');
