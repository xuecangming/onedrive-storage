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
echo "Press Ctrl+C to stop"
echo ""

# Update API URL in index.html if specified
if [ -n "${API_URL}" ]; then
    sed -i "s|window.API_BASE_URL = '.*'|window.API_BASE_URL = '${API_URL}'|g" web-app/index.html
fi

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
