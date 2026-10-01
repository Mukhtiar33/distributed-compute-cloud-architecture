#!/usr/bin/env bash
set -euo pipefail

# Generate CA and Control Plane server certificate for local development.
# In production, these would be issued by a proper PKI.

CERT_DIR="./certs"
mkdir -p "$CERT_DIR"

echo "Generating CA key and certificate..."
openssl genrsa -out "$CERT_DIR/ca.key" 4096 2>/dev/null
openssl req -new -x509 -key "$CERT_DIR/ca.key" -out "$CERT_DIR/ca.crt" \
  -days 3650 -subj "/CN=Distributed Compute Cloud CA" 2>/dev/null

echo "Generating Control Plane server key and certificate..."
openssl genrsa -out "$CERT_DIR/server.key" 2048 2>/dev/null
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" \
  -subj "/CN=localhost" 2>/dev/null

# Create SAN extension for localhost
cat > "$CERT_DIR/server.ext" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

openssl x509 -req -in "$CERT_DIR/server.csr" -CA "$CERT_DIR/ca.crt" \
  -CAkey "$CERT_DIR/ca.key" -CAcreateserial -out "$CERT_DIR/server.crt" \
  -days 365 -extfile "$CERT_DIR/server.ext" 2>/dev/null

echo "Certificates generated in $CERT_DIR/"
echo "  ca.crt - CA certificate (trust anchor)"
echo "  server.crt - Control Plane server certificate"
echo "  server.key - Control Plane server private key"
