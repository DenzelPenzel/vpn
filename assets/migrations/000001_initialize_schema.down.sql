-- Rollback migration: 000001_initialize_schema.down.sql
-- This file reverts the changes made in 000001_initialize_schema.up.sql

-- Drop indexes first
DROP INDEX IF EXISTS idx_user_keys_active;
DROP INDEX IF EXISTS idx_user_keys_server_id;
DROP INDEX IF EXISTS idx_user_keys_user_id;
DROP INDEX IF EXISTS idx_servers_active;
DROP INDEX IF EXISTS idx_servers_location;
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables in reverse order (due to foreign key constraints)
DROP TABLE IF EXISTS user_keys;
DROP TABLE IF EXISTS servers;
DROP TABLE IF EXISTS users;

-- Drop extensions (optional, might be used by other applications)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
