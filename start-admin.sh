#!/bin/bash

# Start the Admin UI
set -e

cd "$(dirname "$0")/admin-ui"

echo "========================================"
echo "  Starting Admin Dashboard..."
echo "========================================"

if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

echo "Starting Vite server..."
npm run dev
