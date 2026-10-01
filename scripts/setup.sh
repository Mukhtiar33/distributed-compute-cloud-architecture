#!/usr/bin/env bash
set -euo pipefail

echo "=== Distributed Compute Cloud — Local Dev Setup ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v go &> /dev/null; then
  echo "ERROR: Go is not installed. Install Go 1.22+ from https://go.dev/dl/"
  exit 1
fi
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "  Go: $GO_VERSION"

if ! command -v node &> /dev/null; then
  echo "ERROR: Node.js is not installed. Install Node 20+ from https://nodejs.org/"
  exit 1
fi
NODE_VERSION=$(node --version)
echo "  Node: $NODE_VERSION"

if ! command -v npm &> /dev/null; then
  echo "ERROR: npm is not installed."
  exit 1
fi
echo "  npm: $(npm --version)"

echo ""
echo "=== All prerequisites met ==="
echo ""

# Build Control Plane
echo "Building Control Plane..."
cd control-plane
go build ./...
cd ..

# Build Worker Agent
echo "Building Worker Agent..."
cd worker-agent
go build ./...
cd ..

# Install website dependencies
echo "Installing website dependencies..."
cd website
npm install
cd ..

echo ""
echo "=== Setup complete ==="
echo ""
echo "To run locally:"
echo "  1. Control Plane:  cd control-plane && go run ./cmd/control-plane"
echo "  2. Worker Agent:   cd worker-agent && go run ./cmd/worker-agent"
echo "  3. Website:        cd website && npm run dev"
echo ""
