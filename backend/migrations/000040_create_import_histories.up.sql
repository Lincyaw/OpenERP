-- Migration: create_import_histories
-- Created: 2026-01-30
-- Description: Create import_histories table for tracking bulk import operations

-- Create enum type for import status
DO $$ BEGIN
    CREATE TYPE import_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create enum type for entity types that can be imported
DO $$ BEGIN
    CREATE TYPE import_entity_type AS ENUM ('products', 'customers', 'suppliers', 'inventory', 'categories');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create import_histories table
CREATE TABLE IF NOT EXISTS import_histories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_type import_entity_type NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL DEFAULT 0,
    total_rows INTEGER NOT NULL DEFAULT 0,
    success_rows INTEGER NOT NULL DEFAULT 0,
    error_rows INTEGER NOT NULL DEFAULT 0,
    skipped_rows INTEGER NOT NULL DEFAULT 0,
    updated_rows INTEGER NOT NULL DEFAULT 0,
    conflict_mode VARCHAR(20) NOT NULL DEFAULT 'skip',
    status import_status NOT NULL DEFAULT 'pending',
    error_details JSONB DEFAULT '[]',
    imported_by UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT chk_import_history_file_size CHECK (file_size >= 0),
    CONSTRAINT chk_import_history_total_rows CHECK (total_rows >= 0),
    CONSTRAINT chk_import_history_success_rows CHECK (success_rows >= 0),
    CONSTRAINT chk_import_history_error_rows CHECK (error_rows >= 0),
    CONSTRAINT chk_import_history_skipped_rows CHECK (skipped_rows >= 0),
    CONSTRAINT chk_import_history_updated_rows CHECK (updated_rows >= 0),
    CONSTRAINT chk_import_history_conflict_mode CHECK (conflict_mode IN ('skip', 'update', 'fail'))
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_import_histories_tenant_id ON import_histories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_histories_entity_type ON import_histories(entity_type);
CREATE INDEX IF NOT EXISTS idx_import_histories_status ON import_histories(status);
CREATE INDEX IF NOT EXISTS idx_import_histories_imported_by ON import_histories(imported_by);
CREATE INDEX IF NOT EXISTS idx_import_histories_started_at ON import_histories(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_histories_completed_at ON import_histories(completed_at DESC);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_import_histories_tenant_status ON import_histories(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_import_histories_tenant_entity_type ON import_histories(tenant_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_import_histories_tenant_started_at ON import_histories(tenant_id, started_at DESC);

-- Create GIN index for JSONB error_details queries
CREATE INDEX IF NOT EXISTS idx_import_histories_error_details ON import_histories USING GIN (error_details);

-- Create trigger for updated_at timestamp
CREATE TRIGGER update_import_histories_updated_at
    BEFORE UPDATE ON import_histories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE import_histories IS 'Audit trail for bulk import operations';
COMMENT ON COLUMN import_histories.entity_type IS 'Type of entity being imported (products, customers, suppliers, inventory, categories)';
COMMENT ON COLUMN import_histories.file_name IS 'Original name of the uploaded CSV file';
COMMENT ON COLUMN import_histories.file_size IS 'Size of the uploaded file in bytes';
COMMENT ON COLUMN import_histories.total_rows IS 'Total number of data rows in the CSV (excluding header)';
COMMENT ON COLUMN import_histories.success_rows IS 'Number of rows successfully imported (new records created)';
COMMENT ON COLUMN import_histories.error_rows IS 'Number of rows that failed validation or import';
COMMENT ON COLUMN import_histories.skipped_rows IS 'Number of rows skipped due to conflict mode';
COMMENT ON COLUMN import_histories.updated_rows IS 'Number of existing rows updated (when conflict_mode=update)';
COMMENT ON COLUMN import_histories.conflict_mode IS 'How to handle conflicts: skip, update, or fail';
COMMENT ON COLUMN import_histories.status IS 'Current status of the import operation';
COMMENT ON COLUMN import_histories.error_details IS 'JSON array of row-level errors with details';
COMMENT ON COLUMN import_histories.imported_by IS 'User who initiated the import';
COMMENT ON COLUMN import_histories.started_at IS 'Timestamp when import processing started';
COMMENT ON COLUMN import_histories.completed_at IS 'Timestamp when import processing finished';
