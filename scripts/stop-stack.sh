#!/usr/bin/env bash
set -euo pipefail

# Stop all running stack processes

echo "Stopping stack..."

# Kill Control Plane
if pgrep -f "control-plane" > /dev/null 2>&1; then
  pkill -f "control-plane"
  echo "Control Plane stopped"
fi

# Kill Website
if pgrep -f "next" > /dev/null 2>&1; then
  pkill -f "next"
  echo "Website stopped"
fi

# Kill Worker Agent
if pgrep -f "worker-agent" > /dev/null 2>&1; then
  pkill -f "worker-agent"
  echo "Worker Agent stopped"
fi

echo "All services stopped."
