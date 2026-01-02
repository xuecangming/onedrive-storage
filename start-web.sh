#!/bin/bash

# Web Application Standalone Server
# This script serves the React web application using Vite dev server

PORT=${WEB_PORT:-5173}
API_URL=${API_URL:-http://localhost:8080}

echo "=========================================="
echo "   OneDrive Cloud Storage - Web Application"
echo "=========================================="
echo ""
echo "Web UI:     http://localhost:${PORT}"
echo "API URL:    ${API_URL}"
echo ""
echo "Note: Make sure the middleware is running at ${API_URL}"
echo ""
echo "Press Ctrl+C to stop"
echo ""

cd cloud-drive

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Start development server
npm run dev -- --port ${PORT}
