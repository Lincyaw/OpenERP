-- Migration: Drop departments table and remove department_id from users
-- Description: Rollback department hierarchy implementation

-- Remove department_id from users
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_user_department;
DROP INDEX IF EXISTS idx_user_department;
ALTER TABLE users DROP COLUMN IF EXISTS department_id;

-- Drop departments table (will cascade to indexes and triggers)
DROP TABLE IF EXISTS departments;
