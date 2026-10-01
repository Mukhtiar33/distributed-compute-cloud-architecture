# Document 18 — Tech Stack Decision

## Objective & Scope
Lock in the technology stack for implementation, so every phase in Document 17 is built consistently, with no per-phase language/framework drift.

## 1. Decision Summary

| Component | Technology | Rationale |
|---|---|---|
| Worker Agent | **Go** | Compiles to a single static binary — no runtime the provider installs. Cross-platform (Windows/Mac/Linux) from one codebase. Native concurrency for simultaneous heartbeat, job execution, and resource-throttling monitoring (Document 08 §3.4). Strong stdlib crypto/TLS support maps directly to Document 01 §3's mTLS requirement. |
| Control Plane | **Go** | Shared language with the Worker Agent — shared protobuf/gRPC definitions, no translation layer, one team/one idiom set. Go's concurrency model fits a scheduler coordinating many concurrent workers well. |
| Agent ↔ Control Plane protocol | **gRPC + mTLS** | Strongly typed, bidirectional streaming (heartbeats, job status updates), built-in TLS. Code-generates client/server stubs directly from `.proto` definitions, so Document 01 §3's auth model and Document 09's job/batch status model map to real, typed code with minimal glue. |
| Consumer-facing API | **REST (HTTP/JSON)**, served as a distinct gateway/service in front of the same Control Plane core | Simpler for a website form to call than gRPC; keeps the consumer-facing trust boundary implementation-separate from the worker-facing one (Document 06 §4). |
| Sandbox mechanism | **Docker, driven via Go's official Docker SDK** | Matches Document 11 §3's container-baseline decision. Uses the industry-standard runtime rather than custom isolation tooling. |
| Durable state (jobs, batches, metering, worker identities) | **PostgreSQL** | ACID transactions required for Document 09 §5's idempotency guarantee (batch status transitions must be atomic, not eventually consistent). |
| Environment Registry storage | **Content-addressed local filesystem in v1**, same interface migrates to S3-compatible object storage later | Matches Document 10 §2's content-addressing model. Migration to real object storage is a config/backend change, not a redesign. |
| Website | **Next.js (React) + Tailwind** | Fast to build download page, upload form, and basic account UI (Document 05, Phase 9). Deployable to Vercel or self-hosted. |
| CI/CD | **GitHub Actions** | Per Document 17's CI/CD design. |

## 2. Repository Structure (binding for all phases)

```
/control-plane        (Go module: api gateway, scheduler, control-plane core, postgres migrations)
/worker-agent         (Go module: agent binary source)
/proto                (shared .proto definitions used by both control-plane and worker-agent)
/website              (Next.js app)
/docs                 (Documents 00-18, kept in sync per Document 00's governing rules)
/.github/workflows    (ci.yml, security-tests.yml, release.yml per Document 17)
```

Every phase's delivered zip must place files under these existing paths — never introduce a new top-level structure (binding rule for all future phase deliveries).

## 3. Why Not Python or Node for the Control Plane
A single-language stack (Go for both agent and control plane) avoids a cross-language gRPC/protobuf translation seam that would otherwise need independent maintenance in two ecosystems. Go's static binary output is also a hard requirement for the Worker Agent (Document 08 §1's "single lightweight install" philosophy) — Python/Node cannot produce a comparable dependency-free binary without significant packaging overhead (PyInstaller, pkg, etc. add real distribution complexity and larger binaries). Using Go for the Control Plane as well removes an unnecessary second stack for a solo builder to maintain.

## 4. Local-to-VPS Consistency
This stack is chosen partly because it collapses the "local laptop as Control Plane" (Document 17 Phase 11) to "real VPS" (Phase 12) migration to a pure configuration change: the same Go binaries and Docker-based sandboxing run identically on a laptop or a VPS, with no interpreter/runtime differences to account for.

## Senior Review
**Risks:** Go has a real learning curve if unfamiliar — mitigated by the fact that Documents 17's phases are small and incremental, so the learning curve is amortized rather than front-loaded.
**Technical Debt:** None — this stack was chosen specifically because it requires no foreseeable rewrite between Pilot and Production scale (Document 16).
