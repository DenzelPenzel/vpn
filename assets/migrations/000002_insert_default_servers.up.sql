-- Migration: 000002_insert_default_servers.up.sql
-- Insert default VPN servers

-- Note: In production, these keys should be replaced with actual WireGuard keys
-- These are placeholder keys for development/testing purposes
INSERT INTO servers (name, location, endpoint, public_key, private_key, port) VALUES
(
    'US-East-1', 
    'New York, USA', 
    'vpn-us-east.example.com', 
    'PLACEHOLDER_PUBLIC_KEY_US_EAST_REPLACE_IN_PRODUCTION', 
    'PLACEHOLDER_PRIVATE_KEY_US_EAST_REPLACE_IN_PRODUCTION', 
    51820
),
(
    'EU-West-1', 
    'London, UK', 
    'vpn-eu-west.example.com', 
    'PLACEHOLDER_PUBLIC_KEY_EU_WEST_REPLACE_IN_PRODUCTION', 
    'PLACEHOLDER_PRIVATE_KEY_EU_WEST_REPLACE_IN_PRODUCTION', 
    51820
),
(
    'Asia-1', 
    'Singapore', 
    'vpn-asia.example.com', 
    'PLACEHOLDER_PUBLIC_KEY_ASIA_REPLACE_IN_PRODUCTION', 
    'PLACEHOLDER_PRIVATE_KEY_ASIA_REPLACE_IN_PRODUCTION', 
    51820
)
ON CONFLICT DO NOTHING;
