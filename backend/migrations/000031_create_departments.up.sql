-- Migration: Create departments table and add department_id to users
-- Description: Implements department hierarchy for department-level data scoping

-- Create departments table
CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    parent_id UUID,
    path VARCHAR(1000) NOT NULL DEFAULT '', -- Materialized path for efficient hierarchy queries
    level INTEGER NOT NULL DEFAULT 0, -- Depth in hierarchy (0 = root)
    sort_order INTEGER NOT NULL DEFAULT 0,
    manager_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    metadata JSONB DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique code per tenant
    CONSTRAINT uq_department_tenant_code UNIQUE (tenant_id, code),

    -- Foreign keys
    CONSTRAINT fk_department_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT fk_department_parent FOREIGN KEY (parent_id) REFERENCES departments(id) ON DELETE RESTRICT,
    CONSTRAINT fk_department_manager FOREIGN KEY (manager_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for common query patterns
CREATE INDEX idx_department_tenant ON departments(tenant_id);
CREATE INDEX idx_department_parent ON departments(parent_id);
CREATE INDEX idx_department_path ON departments(path);
CREATE INDEX idx_department_manager ON departments(manager_id) WHERE manager_id IS NOT NULL;
CREATE INDEX idx_department_status ON departments(tenant_id, status);
CREATE INDEX idx_department_level ON departments(tenant_id, level);

-- Create GIN index for efficient path prefix queries (finding all descendants)
CREATE INDEX idx_department_path_pattern ON departments USING gin (path gin_trgm_ops);

-- Add update trigger for updated_at
CREATE TRIGGER trg_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE departments IS 'Organizational departments for hierarchical data scoping';
COMMENT ON COLUMN departments.code IS 'Unique code within tenant (e.g., SALES, HR)';
COMMENT ON COLUMN departments.path IS 'Materialized path containing ancestor IDs (e.g., /root-id/parent-id/this-id)';
COMMENT ON COLUMN departments.level IS 'Depth in hierarchy tree (0 = root department)';
COMMENT ON COLUMN departments.manager_id IS 'User ID of department manager (can view all department data)';
COMMENT ON COLUMN departments.metadata IS 'Additional key-value metadata';

-- Add department_id column to users table
ALTER TABLE users ADD COLUMN department_id UUID;

-- Add foreign key constraint
ALTER TABLE users ADD CONSTRAINT fk_user_department
    FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL;

-- Create index for users.department_id
CREATE INDEX idx_user_department ON users(department_id) WHERE department_id IS NOT NULL;

-- Add comment for users.department_id
COMMENT ON COLUMN users.department_id IS 'Primary department the user belongs to';

-- Insert default root department for existing tenant
INSERT INTO departments (id, tenant_id, code, name, description, path, level, sort_order, status)
SELECT
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'ROOT',
    'Root Department',
    'Default root department',
    '/00000000-0000-0000-0000-000000000010',
    0,
    0,
    'active'
WHERE NOT EXISTS (
    SELECT 1 FROM departments WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
);
