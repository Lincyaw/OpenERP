-- Migration: Remove Feature Flag permissions (rollback)
-- Description: Removes feature_flag permissions and DEVELOPER role

-- Remove feature_flag permissions from all roles
DELETE FROM role_permissions WHERE resource = 'feature_flag';

-- Remove DEVELOPER role
DELETE FROM roles WHERE code = 'DEVELOPER' AND tenant_id = '00000000-0000-0000-0000-000000000001';
