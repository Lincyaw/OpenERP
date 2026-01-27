-- Migration: Add Feature Flag permissions
-- Description: Adds feature_flag permissions and creates DEVELOPER role
-- Task: FF-BE-008

-- ==============================================
-- STEP 1: Create DEVELOPER role
-- ==============================================
INSERT INTO roles (id, tenant_id, code, name, description, is_system_role, sort_order)
VALUES (
    '00000000-0000-0000-0000-000000000017',
    '00000000-0000-0000-0000-000000000001',
    'DEVELOPER',
    'Developer',
    'Development access for feature flags and testing',
    TRUE,
    8
) ON CONFLICT DO NOTHING;

-- ==============================================
-- STEP 2: Add Feature Flag permissions to ADMIN role
-- ADMIN gets ALL feature flag permissions
-- ==============================================
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000010'::uuid,  -- ADMIN role ID
    '00000000-0000-0000-0000-000000000001'::uuid,  -- Default tenant ID
    perm.code,
    perm.resource,
    perm.action,
    'Admin permission for ' || perm.code
FROM (
    VALUES
        ('feature_flag:read', 'feature_flag', 'read'),
        ('feature_flag:create', 'feature_flag', 'create'),
        ('feature_flag:update', 'feature_flag', 'update'),
        ('feature_flag:delete', 'feature_flag', 'delete'),
        ('feature_flag:override', 'feature_flag', 'override'),
        ('feature_flag:audit', 'feature_flag', 'audit'),
        ('feature_flag:evaluate', 'feature_flag', 'evaluate')
) AS perm(code, resource, action)
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000010'::uuid
    AND rp.code = perm.code
);

-- ==============================================
-- STEP 3: Add Feature Flag permissions to MANAGER role
-- MANAGER gets: read, create, update, override, evaluate
-- ==============================================
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000011'::uuid,  -- MANAGER role ID
    '00000000-0000-0000-0000-000000000001'::uuid,  -- Default tenant ID
    perm.code,
    perm.resource,
    perm.action,
    'Manager permission for ' || perm.code
FROM (
    VALUES
        ('feature_flag:read', 'feature_flag', 'read'),
        ('feature_flag:create', 'feature_flag', 'create'),
        ('feature_flag:update', 'feature_flag', 'update'),
        ('feature_flag:override', 'feature_flag', 'override'),
        ('feature_flag:evaluate', 'feature_flag', 'evaluate')
) AS perm(code, resource, action)
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000011'::uuid
    AND rp.code = perm.code
);

-- ==============================================
-- STEP 4: Add Feature Flag permissions to DEVELOPER role
-- DEVELOPER gets: read, create, update, evaluate
-- ==============================================
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000017'::uuid,  -- DEVELOPER role ID
    '00000000-0000-0000-0000-000000000001'::uuid,  -- Default tenant ID
    perm.code,
    perm.resource,
    perm.action,
    'Developer permission for ' || perm.code
FROM (
    VALUES
        ('feature_flag:read', 'feature_flag', 'read'),
        ('feature_flag:create', 'feature_flag', 'create'),
        ('feature_flag:update', 'feature_flag', 'update'),
        ('feature_flag:evaluate', 'feature_flag', 'evaluate')
) AS perm(code, resource, action)
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000017'::uuid
    AND rp.code = perm.code
);

-- ==============================================
-- STEP 5: Add Feature Flag read permission to other roles
-- Other roles (SALES, PURCHASER, WAREHOUSE, CASHIER, ACCOUNTANT)
-- get: read (view-only) and evaluate (needed to use flags)
-- ==============================================

-- SALES role - feature_flag:read
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000012'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:read',
    'feature_flag',
    'read',
    'Sales permission for feature_flag:read'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000012'::uuid
    AND rp.code = 'feature_flag:read'
);

-- SALES role - feature_flag:evaluate
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000012'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:evaluate',
    'feature_flag',
    'evaluate',
    'Sales permission for feature_flag:evaluate'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000012'::uuid
    AND rp.code = 'feature_flag:evaluate'
);

-- PURCHASER role - feature_flag:read
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000013'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:read',
    'feature_flag',
    'read',
    'Purchaser permission for feature_flag:read'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000013'::uuid
    AND rp.code = 'feature_flag:read'
);

-- PURCHASER role - feature_flag:evaluate
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000013'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:evaluate',
    'feature_flag',
    'evaluate',
    'Purchaser permission for feature_flag:evaluate'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000013'::uuid
    AND rp.code = 'feature_flag:evaluate'
);

-- WAREHOUSE role - feature_flag:read
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000014'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:read',
    'feature_flag',
    'read',
    'Warehouse Staff permission for feature_flag:read'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000014'::uuid
    AND rp.code = 'feature_flag:read'
);

-- WAREHOUSE role - feature_flag:evaluate
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000014'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:evaluate',
    'feature_flag',
    'evaluate',
    'Warehouse Staff permission for feature_flag:evaluate'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000014'::uuid
    AND rp.code = 'feature_flag:evaluate'
);

-- CASHIER role - feature_flag:read
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000015'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:read',
    'feature_flag',
    'read',
    'Cashier permission for feature_flag:read'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000015'::uuid
    AND rp.code = 'feature_flag:read'
);

-- CASHIER role - feature_flag:evaluate
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000015'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:evaluate',
    'feature_flag',
    'evaluate',
    'Cashier permission for feature_flag:evaluate'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000015'::uuid
    AND rp.code = 'feature_flag:evaluate'
);

-- ACCOUNTANT role - feature_flag:read
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000016'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:read',
    'feature_flag',
    'read',
    'Accountant permission for feature_flag:read'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000016'::uuid
    AND rp.code = 'feature_flag:read'
);

-- ACCOUNTANT role - feature_flag:evaluate
INSERT INTO role_permissions (role_id, tenant_id, code, resource, action, description)
SELECT
    '00000000-0000-0000-0000-000000000016'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'feature_flag:evaluate',
    'feature_flag',
    'evaluate',
    'Accountant permission for feature_flag:evaluate'
WHERE NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = '00000000-0000-0000-0000-000000000016'::uuid
    AND rp.code = 'feature_flag:evaluate'
);

-- ==============================================
-- DOCUMENTATION
-- ==============================================

-- Feature Flag Permission Summary:
-- +-----------+------+--------+--------+--------+----------+-------+----------+
-- | Role      | read | create | update | delete | override | audit | evaluate |
-- +-----------+------+--------+--------+--------+----------+-------+----------+
-- | ADMIN     |  ✓   |   ✓    |   ✓    |   ✓    |    ✓     |   ✓   |    ✓     |
-- | MANAGER   |  ✓   |   ✓    |   ✓    |   ✗    |    ✓     |   ✗   |    ✓     |
-- | DEVELOPER |  ✓   |   ✓    |   ✓    |   ✗    |    ✗     |   ✗   |    ✓     |
-- | SALES     |  ✓   |   ✗    |   ✗    |   ✗    |    ✗     |   ✗   |    ✓     |
-- | PURCHASER |  ✓   |   ✗    |   ✗    |   ✗    |    ✗     |   ✗   |    ✓     |
-- | WAREHOUSE |  ✓   |   ✗    |   ✗    |   ✗    |    ✗     |   ✗   |    ✓     |
-- | CASHIER   |  ✓   |   ✗    |   ✗    |   ✗    |    ✗     |   ✗   |    ✓     |
-- | ACCOUNTANT|  ✓   |   ✗    |   ✗    |   ✗    |    ✗     |   ✗   |    ✓     |
-- +-----------+------+--------+--------+--------+----------+-------+----------+
