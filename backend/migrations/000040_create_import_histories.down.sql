-- Migration: create_import_histories (rollback)
-- Created: 2026-01-30
-- Description: Drop import_histories table and related objects

-- Drop the trigger first
DROP TRIGGER IF EXISTS update_import_histories_updated_at ON import_histories;

-- Drop indexes
DROP INDEX IF EXISTS idx_import_histories_error_details;
DROP INDEX IF EXISTS idx_import_histories_tenant_started_at;
DROP INDEX IF EXISTS idx_import_histories_tenant_entity_type;
DROP INDEX IF EXISTS idx_import_histories_tenant_status;
DROP INDEX IF EXISTS idx_import_histories_completed_at;
DROP INDEX IF EXISTS idx_import_histories_started_at;
DROP INDEX IF EXISTS idx_import_histories_imported_by;
DROP INDEX IF EXISTS idx_import_histories_status;
DROP INDEX IF EXISTS idx_import_histories_entity_type;
DROP INDEX IF EXISTS idx_import_histories_tenant_id;

-- Drop the table
DROP TABLE IF EXISTS import_histories;

-- Drop enum types (only if not used elsewhere)
DROP TYPE IF EXISTS import_status;
DROP TYPE IF EXISTS import_entity_type;
