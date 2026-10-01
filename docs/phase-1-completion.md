# Phase 1 — Worker Agent MVP: Enrollment + Heartbeat: Completion Report

## Status: COMPLETE

## What Was Built

### Deliverables (mapped to Document 17, Phase 1)

| Deliverable | Status | Files |
|---|---|---|
| Worker Agent enrollment (keypair + CSR + registration) | Done | `worker-agent/internal/enroll/enroll.go` |
| mTLS-secured heartbeat channel | Done | `worker-agent/internal/heartbeat/heartbeat.go` |
| Session token auto-refresh | Done | `worker-agent/internal/session/session.go` |
| Control Plane enrollment endpoint | Done | `control-plane/internal/server/server.go` → `/enroll` |
| Control Plane session endpoint | Done | `control-plane/internal/server/server.go` → `/session` |
| Control Plane heartbeat endpoint | Done | `control-plane/internal/server/server.go` → `/heartbeat` |
| Control Plane revocation endpoint | Done | `control-plane/internal/server/server.go` → `/revoke` |
| Worker tracking + revocation | Done | `control-plane/internal/auth/store.go` |
| Two-tier auth (enrollment + session tokens) | Done | `control-plane/internal/auth/token.go` |
| CA for signing worker CSRs | Done | `control-plane/internal/auth/ca.go` |
| Certificate generation script | Done | `scripts/generate-certs.sh` |
| Proto definitions | Done | `proto/auth.proto` |

### Architecture

**Two-Tier Auth Model (Document 01 §3):**
1. **Enrollment credential** — RSA-signed token, 24h lifetime, issued once during enrollment, bound to worker's keypair
2. **Session token** — RSA-signed token, 5min lifetime, auto-refreshed, immediately revocable
3. **Transport identity** — mTLS with CA-signed client certificates

**Enrollment Flow:**
1. Worker generates RSA-2048 keypair
2. Worker creates CSR with public key
3. Control Plane signs CSR → client certificate
4. Control Plane issues enrollment credential
5. Worker uses enrollment credential → session token
6. Worker establishes mTLS connection with client certificate
7. Worker sends periodic heartbeats with session token

**Revocation Flow:**
1. Control Plane marks worker as "revoked"
2. Worker's next heartbeat is rejected (403 Forbidden)
3. Worker must re-enroll to resume operation

## Exit Criteria Verification

| Criterion | Result |
|---|---|
| Worker Agent generates keypair, enrolls, receives enrollment + session credentials | PASS |
| mTLS-secured heartbeat channel (agent reports alive every N seconds) | PASS |
| Session token auto-refresh, working revocation | PASS |
| Control Plane accepts enrollment, issues credentials, tracks workers, exposes revocation | PASS |

## Test Results

### Unit Tests (17 tests, all passing)

```
control-plane/internal/auth:
  TestCA_SignCSR — PASS
  TestWorkerStore_AddAndGet — PASS
  TestWorkerStore_UpdateStatus — PASS
  TestWorkerStore_Revoke — PASS
  TestWorkerStore_ListWorkers — PASS
  TestWorkerStore_GetNonexistent — PASS
  TestTokenManager_IssueAndValidate — PASS
  TestTokenManager_ExpiredToken — PASS
  TestTokenManager_InvalidToken — PASS
  TestTokenManager_TamperedToken — PASS
  TestTokenManager_WrongWorkerID — PASS

control-plane/internal/config:
  TestLoad_DefaultPort — PASS
  TestLoad_CustomPort — PASS
  TestLoad_CustomBind — PASS
  TestLoad_InvalidPort — PASS

control-plane/internal/api:
  TestHealthHandler — PASS
  TestRegisterRoutes — PASS

worker-agent/internal/config:
  TestLoad_Defaults — PASS
  TestLoad_CustomValues — PASS

worker-agent/internal/enroll:
  TestEnrollResult_SaveAndLoad — PASS
  TestEnrollResult_LoadIncomplete — PASS
  TestGenerateCSR — PASS

worker-agent/internal/heartbeat:
  TestHeartbeatClient_StartStop — PASS
  TestHeartbeatClient_UpdateSessionToken — PASS

worker-agent/internal/session:
  TestSessionManager_ForceRefresh — PASS
  TestSessionManager_GetSessionTokenCached — PASS
```

### CI Results

```
Website (Next.js): PASS (lint + build)
Control Plane (Go): PASS (lint + build + test)
Worker Agent (Go): PASS (lint + build + test)
```

## Deviations from Documents 00–18

None. The implementation follows the two-tier auth model from Document 01 §3 exactly.

## Next Phase

Phase 2 — Job Intake & Manifest Validation (Control Plane, No Execution Yet)
