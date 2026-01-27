-- Migration: Fix flag_audit_logs table schema
-- Description: Add updated_at, user_agent columns and rename actor_id to user_id

-- Add updated_at column
ALTER TABLE flag_audit_logs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Rename actor_id to user_id (if exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'flag_audit_logs' AND column_name = 'actor_id') THEN
        ALTER TABLE flag_audit_logs RENAME COLUMN actor_id TO user_id;
    END IF;
END $$;

-- Rename actor_ip to ip_address (if exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'flag_audit_logs' AND column_name = 'actor_ip') THEN
        ALTER TABLE flag_audit_logs RENAME COLUMN actor_ip TO ip_address;
    END IF;
END $$;

-- Add user_agent column
ALTER TABLE flag_audit_logs ADD COLUMN IF NOT EXISTS user_agent TEXT;

-- Recreate index with new column name
DROP INDEX IF EXISTS idx_flag_audit_logs_actor;
CREATE INDEX IF NOT EXISTS idx_flag_audit_logs_user_id ON flag_audit_logs(user_id) WHERE user_id IS NOT NULL;
