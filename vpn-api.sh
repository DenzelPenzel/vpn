#!/usr/bin/with-contenv bash

# This script waits for the WireGuard public key and then launches the API server.
PUBLIC_KEY_FILE="/config/publickey"

echo "Waiting for WireGuard public key at ${PUBLIC_KEY_FILE}..."

while [ ! -f "${PUBLIC_KEY_FILE}" ]; do
  sleep 1
done

echo "WireGuard public key found. Starting VPN API service..."

exec /usr/bin/vpn_api
