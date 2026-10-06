# Control Plane Dockerfile
# Multi-stage build: compile Go binary, then create minimal runtime image

FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Control Plane binary
RUN CGO_ENABLED=0 GOOS=linux go build -o control-plane ./cmd/control-plane

# Runtime stage — minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/control-plane .

# Copy certificates (mounted in production)
COPY certs/ ./certs/

# Environment variables (overridden in production)
ENV CONTROL_PLANE_PORT=8443
ENV CONTROL_PLANE_BIND=:8443

EXPOSE 8443

ENTRYPOINT ["./control-plane"]
