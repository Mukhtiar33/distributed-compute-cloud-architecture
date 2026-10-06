#!/usr/bin/env bash
set -euo pipefail

# Phase 12 — VPS Migration Script
# Deploys the Control Plane to a VPS with zero code changes.
# Only configuration changes: host address, TLS certificates, environment variables.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration (override via environment variables)
VPS_HOST="${VPS_HOST:-}"
VPS_USER="${VPS_USER:-root}"
DOMAIN="${DOMAIN:-localhost}"
EMAIL="${EMAIL:-admin@example.com}"

if [ -z "$VPS_HOST" ]; then
  echo "ERROR: VPS_HOST environment variable is required"
  echo "Usage: VPS_HOST=your.vps.ip ./scripts/deploy.sh"
  exit 1
fi

echo "========================================"
echo "Phase 12 — VPS Migration"
echo "========================================"
echo ""
echo "Target VPS: $VPS_USER@$VPS_HOST"
echo "Domain: $DOMAIN"
echo ""

# Step 1: Build locally
echo "Step 1: Building Control Plane..."
cd "$PROJECT_ROOT/control-plane"
go build -o control-plane ./cmd/control-plane

echo "Step 2: Building Website..."
cd "$PROJECT_ROOT/website"
npm run build

# Step 3: Generate self-signed certs for testing (replace with real certs in production)
echo "Step 3: Generating certificates..."
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
    -subj "/CN=$DOMAIN" 2>/dev/null

  cat > certs/server.ext <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

  openssl x509 -req -in certs/server.csr -CA certs/ca.crt \
    -CAkey certs/ca.key -CAcreateserial -out certs/server.crt \
    -days 365 -extfile certs/server.ext 2>/dev/null
fi

# Step 4: Deploy to VPS
echo "Step 4: Deploying to VPS..."
ssh "$VPS_USER@$VPS_HOST" "mkdir -p /opt/distributed-compute-cloud"
rsync -avz --exclude='.git' --exclude='node_modules' --exclude='.next' \
  "$PROJECT_ROOT/" "$VPS_USER@$VPS_HOST:/opt/distributed-compute-cloud/"

# Step 5: Start services on VPS
echo "Step 5: Starting services on VPS..."
ssh "$VPS_USER@$VPS_HOST" <<'REMOTE'
cd /opt/distributed-compute-cloud
docker-compose down 2>/dev/null || true
docker-compose build
docker-compose up -d
REMOTE

echo ""
echo "========================================"
echo "Deployment Complete"
echo "========================================"
echo ""
echo "Control Plane: https://$DOMAIN:8443"
echo "Website: https://$DOMAIN"
echo ""
echo "To check status:"
echo "  ssh $VPS_USER@$VPS_HOST 'cd /opt/distributed-compute-cloud && docker-compose ps'"
echo ""
echo "To view logs:"
echo "  ssh $VPS_USER@$VPS_HOST 'cd /opt/distributed-compute-cloud && docker-compose logs -f'"
echo ""
