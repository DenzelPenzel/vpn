#!/bin/bash

# VPN SaaS Service Setup Script
# This script sets up the development environment

set -e

echo "🔧 Setting up VPN SaaS Service Development Environment"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.24+ first."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | cut -d'o' -f2)
if [[ "$GO_VERSION" < "1.24" ]]; then
    echo "❌ Go version 1.24+ is required. Current version: $GO_VERSION"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from template..."
    cp .env.example .env
    
    # Generate a random JWT secret
    JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
    sed -i.bak "s/your-super-secret-jwt-key-change-in-production-min-32-chars/$JWT_SECRET/" .env
    rm .env.bak 2>/dev/null || true
    
    echo "🔑 Generated random JWT secret"
    echo "⚠️  Please review and update .env file with your settings"
else
    echo "✅ .env file already exists"
fi

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download
go mod tidy

# Create necessary directories
mkdir -p bin
mkdir -p logs
mkdir -p wireguard

echo "🏗️  Building the application..."
go build -o bin/vpn-service ./cmd/server

echo "🐳 Starting services with Docker Compose..."
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 10

# Check if PostgreSQL is ready
until docker-compose exec postgres pg_isready -U vpnadmin -d vpnservice; do
    echo "⏳ Waiting for PostgreSQL..."
    sleep 2
done

echo "✅ PostgreSQL is ready"

echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Review and update .env file if needed"
echo "2. Run 'make docker-up' to start all services"
echo "3. Run 'make test' to run tests"
echo "4. Access the API at http://localhost:8080/api/health"
echo ""
echo "API Documentation: See API_DOCUMENTATION.md"
echo "Development commands: Run 'make help' for available commands"
