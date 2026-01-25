-- Migration: Create users and user_roles tables
-- Description: Creates the user management tables for authentication and authorization

-- Create users table (aggregate root for identity)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(200),
    phone VARCHAR(50),
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(200),
    avatar VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'locked', 'deactivated')),
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    password_changed_at TIMESTAMPTZ,
    must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
    notes TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique username per tenant
    CONSTRAINT uq_user_tenant_username UNIQUE (tenant_id, username),
    -- Ensure unique email per tenant (if provided)
    CONSTRAINT uq_user_tenant_email UNIQUE (tenant_id, email),

    -- Foreign keys
    CONSTRAINT fk_user_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Create indexes for common query patterns
CREATE INDEX idx_user_tenant ON users(tenant_id);
CREATE INDEX idx_user_username ON users(username);
CREATE INDEX idx_user_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_user_status ON users(tenant_id, status);
CREATE INDEX idx_user_last_login ON users(last_login_at);

-- Add update trigger for updated_at
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create user_roles table for many-to-many relationship
CREATE TABLE user_roles (
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, role_id),

    -- Foreign keys
    CONSTRAINT fk_user_role_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_role_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
    -- Note: role_id FK will be added when roles table is created
);

-- Create indexes for user_roles
CREATE INDEX idx_user_role_user ON user_roles(user_id);
CREATE INDEX idx_user_role_role ON user_roles(role_id);
CREATE INDEX idx_user_role_tenant ON user_roles(tenant_id);

-- Add comments for documentation
COMMENT ON TABLE users IS 'System users for authentication and authorization';
COMMENT ON COLUMN users.username IS 'Unique username within tenant for login';
COMMENT ON COLUMN users.email IS 'Email address (optional, must be unique within tenant if provided)';
COMMENT ON COLUMN users.phone IS 'Phone number for contact or 2FA';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.display_name IS 'Human-readable name for display';
COMMENT ON COLUMN users.status IS 'User status: pending, active, locked, deactivated';
COMMENT ON COLUMN users.failed_attempts IS 'Number of consecutive failed login attempts';
COMMENT ON COLUMN users.locked_until IS 'Timestamp until which the user is locked (null = permanent lock)';
COMMENT ON COLUMN users.must_change_password IS 'Whether user must change password on next login';

COMMENT ON TABLE user_roles IS 'Many-to-many relationship between users and roles';

-- Insert a default admin user for development (password: admin123)
-- Note: In production, this should be created through proper user registration
INSERT INTO users (id, tenant_id, username, password_hash, display_name, status)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'admin',
    '$2a$12$awSyzmWliDnUBvJ6tqjs1OnEbpUoOyujmnS67BotFyFIzCCSyFwVW', -- admin123
    'System Administrator',
    'active'
) ON CONFLICT (tenant_id, username) DO NOTHING;
