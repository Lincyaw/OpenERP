-- Migration: Drop stock_takings and stock_taking_items tables

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_stock_taking_items_updated_at ON stock_taking_items;
DROP TRIGGER IF EXISTS trg_stock_takings_updated_at ON stock_takings;

-- Drop tables (items first due to foreign key)
DROP TABLE IF EXISTS stock_taking_items;
DROP TABLE IF EXISTS stock_takings;
