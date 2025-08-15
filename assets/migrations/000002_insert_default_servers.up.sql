-- Migration: 000002_insert_default_servers.up.sql
-- Insert default VPN servers

INSERT INTO servers (id, name, location, endpoint, port) VALUES
(
    'a7f4c3d6-1b3c-4e8b-9f0e-1d2c3b4a5e6f',
    'Default Server',
    'Amazon Linux',
    '35.78.89.198',
    51820
)
ON CONFLICT (id) DO NOTHING;
