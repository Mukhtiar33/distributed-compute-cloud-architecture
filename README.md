# Distributed Compute Cloud

A secure, trustworthy distributed compute cloud platform that transforms idle personal computers into a compute resource.

## Architecture

- **Control Plane** (Go) — Sole coordinator: auth, scanning, scheduling, environment resolution, output validation, metering
- **Worker Agent** (Go) — Runs on provider machines; sandbox lifecycle, environment caching, status reporting
- **Website** (Next.js) — Consumer upload UI and Worker Agent download page
- **Proto** — Shared gRPC/protobuf definitions

## Repository Structure

```
/control-plane        (Go module: api gateway, scheduler, control-plane core)
/worker-agent         (Go module: agent binary source)
/proto                (shared .proto definitions)
/website              (Next.js app)
/docs                 (Architecture documents 00-18)
/.github/workflows    (CI/CD pipelines)
/scripts              (Setup and utility scripts)
```

## Quick Start

```bash
# Clone and setup
git clone <repo-url>
cd distributed-compute-cloud
./scripts/setup.sh

# Run locally (3 terminals)
cd control-plane && go run ./cmd/control-plane
cd worker-agent && go run ./cmd/worker-agent
cd website && npm run dev
```

## Build Plan

See [docs/17-build-plan-phases-cicd.md](docs/17-build-plan-phases-cicd.md) for the 13-phase delivery plan.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Worker Agent | Go (single static binary) |
| Control Plane | Go |
| Agent ↔ Control Plane | gRPC + mTLS |
| Consumer API | REST (HTTP/JSON) |
| Durable State | PostgreSQL |
| Sandboxing | Docker (containers) |
| Website | Next.js + Tailwind |
| CI/CD | GitHub Actions |
