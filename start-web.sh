#!/bin/bash

# Web Application Standalone Server
# This script serves the web application on a separate port from the middleware

PORT=${WEB_PORT:-3000}
API_URL=${API_URL:-http://localhost:8080/api/v1}

echo "=========================================="
echo "   OneDrive Cloud Storage - Web Application"
echo "=========================================="
echo ""
echo "Web UI:     http://localhost:${PORT}"
echo "API URL:    ${API_URL}"
echo ""
echo "Note: Make sure the middleware is running at ${API_URL}"
echo ""
echo "To configure API URL, edit web-app/index.html or set API_URL environment variable"
echo "Press Ctrl+C to stop"
echo ""

cd web-app

# Use Python's built-in HTTP server
if command -v python3 &> /dev/null; then
    python3 -m http.server ${PORT}
elif command -v python &> /dev/null; then
    python -m SimpleHTTPServer ${PORT}
else
    echo "Error: Python is required to run the web server"
    echo "Please install Python or use another HTTP server"
    exit 1
fi
