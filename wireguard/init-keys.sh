#!/bin/bash
set -e

# Directory for WireGuard keys and config
WG_DIR="/config"
PRIVATE_KEY_FILE="$WG_DIR/privatekey"
PUBLIC_KEY_FILE="$WG_DIR/publickey"
CONFIG_FILE="$WG_DIR/wg0.conf"

mkdir -p "$WG_DIR"

# Generate keys only if they don't exist
if [ ! -f "$PRIVATE_KEY_FILE" ]; then
    echo "Private key not found. Generating new keys..."
    wg genkey | tee "$PRIVATE_KEY_FILE"
    chmod 600 "$PRIVATE_KEY_FILE"

    cat "$PRIVATE_KEY_FILE" | wg pubkey | tee "$PUBLIC_KEY_FILE"
    echo "New key pair generated."
else
    echo "Existing private key found. Skipping key generation."
fi

# Always ensure the config file is up-to-date with the correct private key
# This creates the [Interface] section for the server itself.
# The API service will be responsible for adding [Peer] sections later.
echo "Creating/updating wg0.conf..."
cat > "$CONFIG_FILE" <<-EOF
[Interface]
Address = 10.0.0.1/24
ListenPort = 51820
PrivateKey = $(cat $PRIVATE_KEY_FILE)
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
EOF

echo "WireGuard configuration is ready."

# Keep the script running if needed, or pass execution to the main container command
exec "$@"
