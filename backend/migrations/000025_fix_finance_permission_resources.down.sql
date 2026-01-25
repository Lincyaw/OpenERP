-- Rollback migration: Revert finance permission resource names to original
-- This restores 'receivable' and 'payable' resource names

-- Remove the reconcile permissions
DELETE FROM role_permissions
WHERE role_id = '00000000-0000-0000-0000-000000000010'
AND code IN ('account_receivable:reconcile', 'account_payable:reconcile');

-- Revert role_permissions to original resource names
UPDATE role_permissions
SET resource = 'receivable',
    code = REPLACE(code, 'account_receivable:', 'receivable:'),
    description = REPLACE(description, 'account_receivable:', 'receivable:')
WHERE resource = 'account_receivable';

UPDATE role_permissions
SET resource = 'payable',
    code = REPLACE(code, 'account_payable:', 'payable:'),
    description = REPLACE(description, 'account_payable:', 'payable:')
WHERE resource = 'account_payable';
