#!/bin/bash
set -e

# Start Tracker
echo "Starting Tracker..."
nohup node bin/cmd.js -p 8000 > verification/tracker.log 2>&1 &
TRACKER_PID=$!
sleep 5

# Start Web UI
echo "Starting Web UI..."
PORT=3000 TRACKER_URL=ws://localhost:8000 nohup node reference-client/web-server.js > verification/web.log 2>&1 &
WEB_PID=$!
sleep 5

# Run Playwright
echo "Running Playwright Verification..."
python3 verification/verify_remote.py

# Cleanup
echo "Cleaning up..."
kill $TRACKER_PID
kill $WEB_PID
echo "Verification Done."
