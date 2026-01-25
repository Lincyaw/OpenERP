-- Migration: Fix finance permission resource names
-- The original migration used 'receivable' and 'payable' but the domain model
-- and frontend expect 'account_receivable' and 'account_payable'.
-- This migration updates the resource names to match the expected values.

-- Update role_permissions to use correct resource names
UPDATE role_permissions
SET resource = 'account_receivable',
    code = REPLACE(code, 'receivable:', 'account_receivable:'),
    description = REPLACE(description, 'receivable:', 'account_receivable:')
WHERE resource = 'receivable';

UPDATE role_permissions
SET resource = 'account_payable',
    code = REPLACE(code, 'payable:', 'account_payable:'),
    description = REPLACE(description, 'payable:', 'account_payable:')
WHERE resource = 'payable';

-- Also add reconcile permissions for account_receivable and account_payable
-- (these are needed for the finance reconciliation features)
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    perm.code,
    perm.resource,
    perm.action,
    'Admin permission for ' || perm.code
FROM (
    VALUES
        ('account_receivable:reconcile', 'account_receivable', 'reconcile'),
        ('account_payable:reconcile', 'account_payable', 'reconcile')
) AS perm(code, resource, action)
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000010'
    AND rp.code = perm.code
);
