-- Migration: Drop users and user_roles tables
DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS users;
