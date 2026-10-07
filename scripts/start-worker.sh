#!/usr/bin/env bash
set -euo pipefail

# Start a Worker Agent on a remote machine, connecting to the laptop-as-VPS.
# Usage: ./scripts/start-worker.sh <worker-id> <laptop-ip>

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

WORKER_ID="${1:-}"
LAPTOP_IP="${2:-}"

if [ -z "$WORKER_ID" ] || [ -z "$LAPTOP_IP" ]; then
  echo "Usage: $0 <worker-id> <laptop-ip>"
  echo "Example: $0 worker-1 192.168.1.100"
  exit 1
fi

echo "Starting Worker Agent..."
echo "  Worker ID: $WORKER_ID"
echo "  Control Plane: https://$LAPTOP_IP:8443"

cd "$PROJECT_ROOT/worker-agent"

# Build if binary doesn't exist
if [ ! -f worker-agent ]; then
  echo "Building Worker Agent..."
  go build -o worker-agent ./cmd/worker-agent
fi

WORKER_ID=$WORKER_ID \
WORKER_CONTROL_PLANE_URL="https://$LAPTOP_IP:8443" \
WORKER_CREDENTIALS_DIR="$PROJECT_ROOT/.worker-credentials" \
./worker-agent
