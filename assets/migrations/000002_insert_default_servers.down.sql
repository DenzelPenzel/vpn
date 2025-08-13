-- Rollback migration: 000002_insert_default_servers.down.sql
-- Remove default servers

DELETE FROM servers WHERE name IN ('US-East-1', 'EU-West-1', 'Asia-1');
