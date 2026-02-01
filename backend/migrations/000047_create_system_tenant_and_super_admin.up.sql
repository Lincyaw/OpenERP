-- Migration: create_system_tenant_and_super_admin
-- Created: 2026-02-01
-- Description: Creates system tenant and super admin role for cross-tenant management
-- Task: P3-ADMIN-001

-- ============================================================================
-- SYSTEM TENANT
-- ============================================================================
-- The system tenant (ID: 00000000-0000-0000-0000-000000000000) is a special tenant
-- used for super admin operations. It should never be used for normal business
-- operations and should be excluded from normal tenant queries.

INSERT INTO tenants (
    id,
    name,
    code,
    status,
    plan,
    contact_name,
    contact_email,
    notes,
    config_max_users,
    config_max_warehouses,
    config_max_products
)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    'System Tenant',
    'SYSTEM',
    'active',
    'enterprise',
    'System Administrator',
    'admin@system.local',
    'System tenant for super admin operations. DO NOT use for business operations.',
    9999,
    9999,
    999999
) ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    code = EXCLUDED.code,
    status = EXCLUDED.status,
    plan = EXCLUDED.plan,
    notes = EXCLUDED.notes;

-- Add comment to identify system tenant
COMMENT ON COLUMN tenants.id IS 'Tenant UUID. ID 00000000-0000-0000-0000-000000000000 is reserved for system tenant.';

-- ============================================================================
-- SUPER ADMIN ROLE
-- ============================================================================
-- The super admin role has full access to all tenant management operations.
-- This role is only available in the system tenant.

INSERT INTO roles (
    id,
    tenant_id,
    code,
    name,
    description,
    is_system_role,
    is_enabled,
    sort_order
)
VALUES (
    '00000000-0000-0000-0000-000000000100',
    '00000000-0000-0000-0000-000000000000',
    'SUPER_ADMIN',
    'Super Administrator',
    'Full cross-tenant management access. Can manage all tenants, users, and system settings.',
    TRUE,
    TRUE,
    0
) ON CONFLICT (tenant_id, code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_system_role = EXCLUDED.is_system_role;

-- ============================================================================
-- SUPER ADMIN PERMISSIONS
-- ============================================================================
-- Grant all tenant management permissions to super admin role

-- Tenant management permissions
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
VALUES
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant:read', 'tenant', 'read', 'View tenant details and list all tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant:create', 'tenant', 'create', 'Create new tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant:update', 'tenant', 'update', 'Update tenant information'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant:delete', 'tenant', 'delete', 'Delete tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant:suspend', 'tenant', 'suspend', 'Suspend and reactivate tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant:manage', 'tenant', 'manage', 'Full tenant management access')
ON CONFLICT (role_id, code) DO UPDATE SET
    description = EXCLUDED.description;

-- User management permissions (cross-tenant)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
VALUES
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'user:read', 'user', 'read', 'View users across all tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'user:create', 'user', 'create', 'Create users in any tenant'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'user:update', 'user', 'update', 'Update users in any tenant'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'user:delete', 'user', 'delete', 'Delete users in any tenant'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'user:manage', 'user', 'manage', 'Full user management access')
ON CONFLICT (role_id, code) DO UPDATE SET
    description = EXCLUDED.description;

-- Role management permissions (cross-tenant)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
VALUES
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'role:read', 'role', 'read', 'View roles across all tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'role:create', 'role', 'create', 'Create roles in any tenant'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'role:update', 'role', 'update', 'Update roles in any tenant'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'role:delete', 'role', 'delete', 'Delete roles in any tenant'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'role:manage', 'role', 'manage', 'Full role management access')
ON CONFLICT (role_id, code) DO UPDATE SET
    description = EXCLUDED.description;

-- ============================================================================
-- DEFAULT SUPER ADMIN USER
-- ============================================================================
-- Create a default super admin user. Password should be changed immediately after first login.
-- Default password: superadmin123 (bcrypt hash)
-- In production, this should be configured via environment variables or created through
-- a secure initialization process.

INSERT INTO users (
    id,
    tenant_id,
    username,
    email,
    password_hash,
    display_name,
    status,
    must_change_password,
    notes
)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000000',
    'superadmin',
    'superadmin@system.local',
    '$2a$12$gSlssaFxGQMH59e4McH.KuCVa6WyC1RHAUmKvtwP0Q.LxWF8NpObO', -- superadmin123
    'Super Administrator',
    'active',
    TRUE,
    'Default super admin user. Change password immediately after first login.'
) ON CONFLICT (tenant_id, username) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    notes = EXCLUDED.notes;

-- Assign super admin role to the default super admin user
INSERT INTO user_roles (user_id, role_id, tenant_id)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000100',
    '00000000-0000-0000-0000-000000000000'
) ON CONFLICT (user_id, role_id) DO NOTHING;

-- ============================================================================
-- DATA SCOPE FOR SUPER ADMIN
-- ============================================================================
-- Super admin has 'all' data scope for all resources (cross-tenant access)

INSERT INTO role_data_scopes (role_id, tenant_id, resource, scope_type, description)
VALUES
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'tenant', 'all', 'Access all tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'user', 'all', 'Access all users across tenants'),
    ('00000000-0000-0000-0000-000000000100', '00000000-0000-0000-0000-000000000000', 'role', 'all', 'Access all roles across tenants')
ON CONFLICT (role_id, resource) DO UPDATE SET
    scope_type = EXCLUDED.scope_type,
    description = EXCLUDED.description;

-- Add comments for documentation
COMMENT ON TABLE roles IS 'RBAC roles for authorization. Role ID 00000000-0000-0000-0000-000000000100 is the super admin role.';
