-- Migration: Create roles, role_permissions, and role_data_scopes tables
-- Description: Creates the role management tables for RBAC authorization

-- Create roles table (aggregate root for roles)
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system_role BOOLEAN NOT NULL DEFAULT FALSE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique code per tenant
    CONSTRAINT uq_role_tenant_code UNIQUE (tenant_id, code),

    -- Foreign keys
    CONSTRAINT fk_role_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for roles
CREATE INDEX idx_role_tenant ON roles(tenant_id);
CREATE INDEX idx_role_code ON roles(code);
CREATE INDEX idx_role_enabled ON roles(tenant_id, is_enabled);
CREATE INDEX idx_role_system ON roles(tenant_id, is_system_role);

-- Add update trigger for roles
CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create role_permissions table for storing role permissions
CREATE TABLE role_permissions (
    role_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    code VARCHAR(100) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (role_id, code),

    -- Foreign keys
    CONSTRAINT fk_role_permission_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permission_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for role_permissions
CREATE INDEX idx_role_permission_role ON role_permissions(role_id);
CREATE INDEX idx_role_permission_tenant ON role_permissions(tenant_id);
CREATE INDEX idx_role_permission_resource ON role_permissions(resource);
CREATE INDEX idx_role_permission_code ON role_permissions(code);

-- Create role_data_scopes table for data-level authorization
CREATE TABLE role_data_scopes (
    role_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    resource VARCHAR(50) NOT NULL,
    scope_type VARCHAR(20) NOT NULL CHECK (scope_type IN ('all', 'self', 'department', 'custom')),
    scope_values TEXT, -- JSON array for custom scopes
    description VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (role_id, resource),

    -- Foreign keys
    CONSTRAINT fk_role_data_scope_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_data_scope_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for role_data_scopes
CREATE INDEX idx_role_data_scope_role ON role_data_scopes(role_id);
CREATE INDEX idx_role_data_scope_tenant ON role_data_scopes(tenant_id);

-- Add FK constraint to user_roles table referencing roles
ALTER TABLE user_roles
    ADD CONSTRAINT fk_user_role_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;

-- Add comments for documentation
COMMENT ON TABLE roles IS 'RBAC roles for authorization';
COMMENT ON COLUMN roles.code IS 'Unique role code within tenant (e.g., ADMIN, MANAGER)';
COMMENT ON COLUMN roles.name IS 'Human-readable role name';
COMMENT ON COLUMN roles.is_system_role IS 'System roles cannot be deleted';
COMMENT ON COLUMN roles.is_enabled IS 'Disabled roles do not grant permissions';
COMMENT ON COLUMN roles.sort_order IS 'Display ordering for roles';

COMMENT ON TABLE role_permissions IS 'Functional permissions granted to roles';
COMMENT ON COLUMN role_permissions.code IS 'Permission code in format resource:action (e.g., product:create)';
COMMENT ON COLUMN role_permissions.resource IS 'Resource being protected (e.g., product, customer)';
COMMENT ON COLUMN role_permissions.action IS 'Action being permitted (e.g., create, read, update, delete)';

COMMENT ON TABLE role_data_scopes IS 'Data-level authorization scopes for roles';
COMMENT ON COLUMN role_data_scopes.scope_type IS 'Type of data scope: all, self, department, or custom';
COMMENT ON COLUMN role_data_scopes.scope_values IS 'JSON array of scope values for custom scopes';

-- Insert default system roles for the default tenant
INSERT INTO roles (id, tenant_id, code, name, description, is_system_role, sort_order)
VALUES
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'ADMIN', 'System Administrator', 'Full system access', TRUE, 1),
    ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'MANAGER', 'Manager', 'Management access', TRUE, 2),
    ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'SALES', 'Sales', 'Sales operations access', TRUE, 3),
    ('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', 'PURCHASER', 'Purchaser', 'Purchase operations access', TRUE, 4),
    ('00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', 'WAREHOUSE', 'Warehouse Staff', 'Warehouse operations access', TRUE, 5),
    ('00000000-0000-0000-0000-000000000015', '00000000-0000-0000-0000-000000000001', 'CASHIER', 'Cashier', 'Payment operations access', TRUE, 6),
    ('00000000-0000-0000-0000-000000000016', '00000000-0000-0000-0000-000000000001', 'ACCOUNTANT', 'Accountant', 'Finance operations access', TRUE, 7)
ON CONFLICT DO NOTHING;

-- Insert admin permissions for the ADMIN role
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    r.resource || ':' || a.action,
    r.resource,
    a.action,
    'Admin permission for ' || r.resource || ':' || a.action
FROM (
    SELECT UNNEST(ARRAY['product', 'category', 'customer', 'supplier', 'warehouse', 'inventory',
                        'sales_order', 'purchase_order', 'sales_return', 'purchase_return',
                        'account_receivable', 'account_payable', 'receipt', 'payment', 'expense', 'income',
                        'report', 'user', 'role', 'tenant']) AS resource
) r
CROSS JOIN (
    SELECT UNNEST(ARRAY['create', 'read', 'update', 'delete', 'admin']) AS action
) a
ON CONFLICT DO NOTHING;

-- Insert special inventory permissions for ADMIN role (adjust, lock, unlock)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
VALUES
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'inventory:adjust', 'inventory', 'adjust', 'Admin permission for inventory:adjust'),
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'inventory:lock', 'inventory', 'lock', 'Admin permission for inventory:lock'),
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'inventory:unlock', 'inventory', 'unlock', 'Admin permission for inventory:unlock'),
    -- Finance reconcile permissions
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'account_receivable:reconcile', 'account_receivable', 'reconcile', 'Admin permission for account_receivable:reconcile'),
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'account_payable:reconcile', 'account_payable', 'reconcile', 'Admin permission for account_payable:reconcile')
ON CONFLICT DO NOTHING;

-- Assign admin role to the default admin user
INSERT INTO user_roles (user_id, role_id, tenant_id)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001'
) ON CONFLICT DO NOTHING;
