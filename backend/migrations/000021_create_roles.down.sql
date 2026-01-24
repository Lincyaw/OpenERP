-- Migration: Drop roles, role_permissions, and role_data_scopes tables
-- Description: Drops the role management tables for RBAC authorization

-- Remove default role assignments
DELETE FROM user_roles WHERE role_id IN (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000012',
    '00000000-0000-0000-0000-000000000013',
    '00000000-0000-0000-0000-000000000014',
    '00000000-0000-0000-0000-000000000015',
    '00000000-0000-0000-0000-000000000016'
);

-- Remove FK constraint from user_roles
ALTER TABLE user_roles
    DROP CONSTRAINT IF EXISTS fk_user_role_role;

-- Drop role_data_scopes table
DROP TABLE IF EXISTS role_data_scopes;

-- Drop role_permissions table
DROP TABLE IF EXISTS role_permissions;

-- Drop roles table
DROP TABLE IF EXISTS roles;
