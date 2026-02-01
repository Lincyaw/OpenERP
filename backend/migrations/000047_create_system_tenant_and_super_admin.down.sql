-- Migration: create_system_tenant_and_super_admin (DOWN)
-- Created: 2026-02-01
-- Description: Removes system tenant and super admin role
-- Task: P3-ADMIN-001

-- Remove user role assignment first
DELETE FROM user_roles
WHERE user_id = '00000000-0000-0000-0000-000000000101'
  AND role_id = '00000000-0000-0000-0000-000000000100';

-- Remove super admin user
DELETE FROM users
WHERE id = '00000000-0000-0000-0000-000000000101'
  AND tenant_id = '00000000-0000-0000-0000-000000000000';

-- Remove data scopes for super admin role
DELETE FROM role_data_scopes
WHERE role_id = '00000000-0000-0000-0000-000000000100'
  AND tenant_id = '00000000-0000-0000-0000-000000000000';

-- Remove permissions for super admin role
DELETE FROM role_permissions
WHERE role_id = '00000000-0000-0000-0000-000000000100'
  AND tenant_id = '00000000-0000-0000-0000-000000000000';

-- Remove super admin role
DELETE FROM roles
WHERE id = '00000000-0000-0000-0000-000000000100'
  AND tenant_id = '00000000-0000-0000-0000-000000000000';

-- Remove system tenant
DELETE FROM tenants
WHERE id = '00000000-0000-0000-0000-000000000000';
