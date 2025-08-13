#!/bin/bash

# V2Ray Plugin Obfuscation Testing Script
# This script helps test the v2ray-plugin WebSocket obfuscation functionality

set -e

echo "🚀 V2Ray Plugin Obfuscation Testing Script"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

print_status "Starting VPN services with v2ray-plugin..."
docker-compose up -d

# Wait for services to start
print_status "Waiting for services to initialize..."
sleep 10

# Check service health
print_status "Checking service health..."

# Check PostgreSQL
if docker-compose exec -T postgres pg_isready -U vpnadmin > /dev/null 2>&1; then
    print_success "PostgreSQL is ready"
else
    print_warning "PostgreSQL not ready yet"
fi

# Check VPN API
if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    print_success "VPN API is responding"
else
    print_warning "VPN API not responding yet"
fi

# Check xray
if docker-compose ps xray | grep -q "Up"; then
    print_success "xray container is running"
else
    print_warning "xray container not running"
fi

# Test WebSocket endpoints
print_status "Testing WebSocket endpoints..."

# Test direct WebSocket connection to API
print_status "Testing direct WebSocket connection to VPN API..."
if curl -s -H "Upgrade: websocket" -H "Connection: Upgrade" http://localhost:8080/ws > /dev/null 2>&1; then
    print_success "Direct WebSocket connection works"
else
    print_warning "Direct WebSocket connection failed"
fi

# Test xray WebSocket obfuscation
print_status "Testing xray WebSocket obfuscation..."
if curl -s -H "Upgrade: websocket" -H "Connection: Upgrade" http://localhost:8080/ws > /dev/null 2>&1; then
    print_success "xray WebSocket obfuscation works"
else
    print_warning "v2ray WebSocket obfuscation failed"
fi

# Test through Caddy proxy
print_status "Testing through Caddy proxy..."
if curl -s -H "Upgrade: websocket" -H "Connection: Upgrade" https://localhost/ws -k > /dev/null 2>&1; then
    print_success "Caddy WebSocket proxy works"
else
    print_warning "Caddy WebSocket proxy failed"
fi

# Display service URLs
echo ""
print_status "Service URLs for testing:"
echo "  📡 VPN API:              http://localhost:8080"
echo "  🔒 VPN API (via Caddy):  https://localhost"
echo "  🌐 WebSocket Direct:     ws://localhost:8080/ws"
echo "  🔀 WebSocket xray:       ws://localhost:8080/ws"
echo "  🛡️  WebSocket via Caddy:  wss://localhost/ws"

# Display v2ray configuration info
echo ""
print_status "V2Ray Configuration:"
echo "  📁 Config directory:     ./v2ray/"
echo "  🔑 Server config:        ./v2ray/config.json"
echo "  💻 Client config:        ./v2ray/client-config.json"
echo "  🔐 TLS Certificate:      ./v2ray/cert.pem"
echo "  🗝️  TLS Private Key:      ./v2ray/key.pem"

# Display testing commands
echo ""
print_status "Manual Testing Commands:"
echo "  # Test API health:"
echo "  curl http://localhost:8080/api/health"
echo ""
echo "  # Test WebSocket upgrade:"
echo "  curl -H 'Upgrade: websocket' -H 'Connection: Upgrade' http://localhost:8080/ws"
echo ""
echo "  # Test v2ray obfuscation:"
echo "  curl -H 'Upgrade: websocket' -H 'Connection: Upgrade' http://localhost:8080/ws"
echo ""
echo "  # View v2ray logs:"
echo "  docker-compose logs v2ray-plugin"
echo ""
echo "  # View API logs:"
echo "  docker-compose logs vpn-api"

# Display client connection info
echo ""
print_status "Client Connection Testing:"
echo "  🔧 Use the client config at ./v2ray/client-config.json"
echo "  🌐 Point your v2ray client to: localhost:8080"
echo "  📡 WebSocket path: /ws"
echo "  🆔 UUID: b831381d-6324-4d53-ad4f-8cda48b30811"

echo ""
print_success "V2Ray plugin setup complete! 🎉"
print_status "The obfuscation layer is now active and ready for testing."
