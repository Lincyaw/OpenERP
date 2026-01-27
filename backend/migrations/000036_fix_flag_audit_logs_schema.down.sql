-- Rollback: Revert flag_audit_logs table schema changes

-- Drop user_agent column
ALTER TABLE flag_audit_logs DROP COLUMN IF EXISTS user_agent;

-- Rename ip_address back to actor_ip
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'flag_audit_logs' AND column_name = 'ip_address') THEN
        ALTER TABLE flag_audit_logs RENAME COLUMN ip_address TO actor_ip;
    END IF;
END $$;

-- Rename user_id back to actor_id
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'flag_audit_logs' AND column_name = 'user_id') THEN
        ALTER TABLE flag_audit_logs RENAME COLUMN user_id TO actor_id;
    END IF;
END $$;

-- Drop updated_at column
ALTER TABLE flag_audit_logs DROP COLUMN IF EXISTS updated_at;

-- Recreate index with original name
DROP INDEX IF EXISTS idx_flag_audit_logs_user_id;
CREATE INDEX IF NOT EXISTS idx_flag_audit_logs_actor ON flag_audit_logs(actor_id) WHERE actor_id IS NOT NULL;
