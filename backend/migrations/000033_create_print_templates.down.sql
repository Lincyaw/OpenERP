-- Migration: create_print_templates (down)
-- Created: 2026-01-27
-- Description: Drop print_templates and print_jobs tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_print_jobs_updated_at ON print_jobs;
DROP TRIGGER IF EXISTS update_print_templates_updated_at ON print_templates;

-- Drop print_jobs table first (depends on print_templates)
DROP TABLE IF EXISTS print_jobs;

-- Drop print_templates table
DROP TABLE IF EXISTS print_templates;
