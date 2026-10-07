#!/usr/bin/env bash
set -euo pipefail

# Start the full stack without Docker — direct processes on your laptop.
# Control Plane (Go binary) + Website (Next.js) + optional Worker Agent.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
LAPTOP_IP=$(hostname -I | awk '{print $1}')
CP_PORT="${CP_PORT:-8443}"
WEBSITE_PORT="${WEBSITE_PORT:-3000}"
WORKER_ID="${WORKER_ID:-}"

echo "========================================"
echo "Laptop-as-VPS: Start Stack (No Docker)"
echo "========================================"
echo ""
echo "Laptop IP: $LAPTOP_IP"
echo "Control Plane: https://$LAPTOP_IP:$CP_PORT"
echo "Website: http://$LAPTOP_IP:$WEBSITE_PORT"
echo ""

# Build Control Plane
echo "Building Control Plane..."
cd "$PROJECT_ROOT/control-plane"
go build -o control-plane ./cmd/control-plane

# Build Worker Agent
echo "Building Worker Agent..."
cd "$PROJECT_ROOT/worker-agent"
go build -o worker-agent ./cmd/worker-agent

# Generate certificates
echo "Generating certificates..."
cd "$PROJECT_ROOT"
mkdir -p certs

if [ ! -f certs/ca.key ]; then
  openssl genrsa -out certs/ca.key 4096 2>/dev/null
  openssl req -new -x509 -key certs/ca.key -out certs/ca.crt \
    -days 3650 -subj "/CN=Distributed Compute Cloud CA" 2>/dev/null
fi

if [ ! -f certs/server.key ]; then
  openssl genrsa -out certs/server.key 2048 2>/dev/null
  openssl req -new -key certs/server.key -out certs/server.csr \
    -subj "/CN=$LAPTOP_IP" 2>/dev/null

  cat > certs/server.ext <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
IP.1 = $LAPTOP_IP
DNS.1 = localhost
IP.2 = 127.0.0.1
EOF

  openssl x509 -req -in certs/server.csr -CA certs/ca.crt \
    -CAkey certs/ca.key -CAcreateserial -out certs/server.crt \
    -days 365 -extfile certs/server.ext 2>/dev/null
fi

# Start Control Plane
echo "Starting Control Plane..."
cd "$PROJECT_ROOT"
CONTROL_PLANE_PORT=$CP_PORT \
CONTROL_PLANE_BIND=:$CP_PORT \
nohup ./control-plane/control-plane > /tmp/control-plane.log 2>&1 &
CP_PID=$!
echo "Control Plane started (PID: $CP_PID)"

# Wait for Control Plane
echo "Waiting for Control Plane to be ready..."
for i in {1..30}; do
  if curl -sk "https://localhost:$CP_PORT/health" > /dev/null 2>&1; then
    echo "Control Plane is ready!"
    break
  fi
  sleep 1
done

# Start Website
echo "Starting Website..."
cd "$PROJECT_ROOT/website"
CONTROL_PLANE_URL="https://localhost:$CP_PORT" \
nohup npm start > /tmp/website.log 2>&1 &
WEBSITE_PID=$!
echo "Website started (PID: $WEBSITE_PID)"

# Start Worker Agent (if WORKER_ID is set)
if [ -n "$WORKER_ID" ]; then
  echo "Starting Worker Agent..."
  cd "$PROJECT_ROOT/worker-agent"
  WORKER_ID=$WORKER_ID \
  WORKER_CONTROL_PLANE_URL="https://localhost:$CP_PORT" \
  WORKER_CREDENTIALS_DIR="$PROJECT_ROOT/.worker-credentials" \
  nohup ./worker-agent > /tmp/worker-agent.log 2>&1 &
  WORKER_PID=$!
  echo "Worker Agent started (PID: $WORKER_PID)"
fi

echo ""
echo "========================================"
echo "Stack Started Successfully!"
echo "========================================"
echo ""
echo "Control Plane: https://$LAPTOP_IP:$CP_PORT"
echo "Website: http://$LAPTOP_IP:$WEBSITE_PORT"
echo ""
echo "Logs:"
echo "  Control Plane: /tmp/control-plane.log"
echo "  Website: /tmp/website.log"
if [ -n "$WORKER_ID" ]; then
  echo "  Worker Agent: /tmp/worker-agent.log"
fi
echo ""
echo "To stop: ./scripts/stop-stack.sh"
echo ""
