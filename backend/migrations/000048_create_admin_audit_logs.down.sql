-- Migration: create_admin_audit_logs (down)
-- Created: 2026-02-01
-- Description: Drops admin audit logs table
-- Task: P3-ADMIN-003

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_prevent_audit_log_delete ON admin_audit_logs;
DROP TRIGGER IF EXISTS trg_prevent_audit_log_update ON admin_audit_logs;

-- Drop the function
DROP FUNCTION IF EXISTS prevent_audit_log_modification();

-- Drop the table (this will also drop all indexes and constraints)
DROP TABLE IF EXISTS admin_audit_logs;
