#!/usr/bin/with-contenv bash
# This script is run by s6-overlay as a service.

echo "Starting VPN API service..."

# Execute the vpn_api binary
# The logs will be automatically handled by s6-overlay
exec /usr/bin/vpn_api
