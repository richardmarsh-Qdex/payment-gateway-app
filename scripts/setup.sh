#!/bin/bash

# Payment Gateway Setup Script

set -e

echo "🚀 Setting up Payment Gateway..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.23 or higher."
    exit 1
fi

echo "✅ Go is installed: $(go version)"

# Check if PostgreSQL is running
if ! command -v psql &> /dev/null; then
    echo "⚠️  PostgreSQL client not found. Make sure PostgreSQL is installed and running."
else
    echo "✅ PostgreSQL client found"
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from .env.example..."
    cp .env.example .env
    echo "⚠️  Please update .env with your configuration"
else
    echo "✅ .env file already exists"
fi

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download
go mod tidy

# Create database if it doesn't exist (requires PostgreSQL to be running)
if command -v psql &> /dev/null; then
    echo "🗄️  Checking database..."
    PGPASSWORD=${DB_PASSWORD:-postgres} psql -h ${DB_HOST:-localhost} -U ${DB_USER:-postgres} -tc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME:-payment_gateway}'" | grep -q 1 || \
    PGPASSWORD=${DB_PASSWORD:-postgres} psql -h ${DB_HOST:-localhost} -U ${DB_USER:-postgres} -c "CREATE DATABASE ${DB_NAME:-payment_gateway};" && \
    echo "✅ Database created" || echo "⚠️  Could not create database. Please create it manually."
fi

# Build the application
echo "🔨 Building application..."
go build -o payment-gateway main.go

echo ""
echo "✅ Setup complete!"
echo ""
echo "To run the application:"
echo "  ./payment-gateway"
echo ""
echo "Or:"
echo "  go run main.go"
echo ""

