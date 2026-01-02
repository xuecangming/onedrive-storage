#!/bin/bash

# Start both Backend and Frontend
set -e

# Function to kill background processes on exit
cleanup() {
    echo "Stopping services..."
    kill $(jobs -p) 2>/dev/null
}
trap cleanup EXIT

echo "========================================"
echo "  Starting OneDrive Storage System"
echo "========================================"

# Start Backend in background
echo "[1/2] Starting Backend Server..."
./start.sh &
BACKEND_PID=$!

# Wait for backend to be ready
echo "Waiting for backend to initialize..."
sleep 5

# Start Frontend
echo "[2/2] Starting Admin UI..."
cd admin-ui
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install
fi
npm run dev

# Wait for backend process
wait $BACKEND_PID
