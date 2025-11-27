#!/bin/bash

# OneDrive Storage Middleware - Quick Start Script
set -e

cd "$(dirname "$0")"

echo "========================================"
echo "  OneDrive Storage Middleware"
echo "========================================"

# Start PostgreSQL
echo "[1/3] Starting PostgreSQL..."
docker compose up -d
for i in {1..30}; do
    docker exec onedrive-storage-db pg_isready -U postgres &>/dev/null && break
    sleep 2
done

# Build
echo "[2/3] Building..."
go mod tidy
go build -o bin/server cmd/server/main.go

# Start
echo "[3/3] Starting server..."
echo ""
echo "  API Base URL:  http://localhost:8080/api/v1"
echo ""
echo "  Endpoints:"
echo "    Health:    GET  /api/v1/health"
echo "    Buckets:   GET  /api/v1/buckets"
echo "    Objects:   GET  /api/v1/objects/{bucket}"
echo "    Accounts:  GET  /api/v1/accounts"
echo "    OAuth:     GET  /api/v1/oauth/setup"
echo ""

export DB_PASSWORD=postgres123
./bin/server
