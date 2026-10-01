# Document 08 — Worker Agent Philosophy & Lifecycle

## Objective & Scope
Specify the Worker Agent's full lifecycle, from download through steady-state operation to decommission, and the philosophy governing its design.

## 1. Core Classification
| Component | Classification |
|---|---|
| Single-binary install, background service | CORE |
| Environment caching/provisioning | CORE |
| Sandbox lifecycle management | CORE |
| Resource throttling (existence of) | CORE |
| Resource throttling (precision/sophistication) | SCALE PARAMETER |
| Status reporting | CORE |

## 2. Philosophy
The Worker Agent is a guest on someone else's machine. Every design decision is filtered through: *does this respect the provider's ownership of their own computer?* This means no silent background resource hogging, no unremovable state, no unexplained network activity, and a clean uninstall that leaves nothing behind.

## 3. Lifecycle Stages

### 3.1 Download & Enrollment
- Provider downloads the agent from the platform website.
- First run triggers enrollment: generates a local keypair, registers with the Control Plane, receives an enrollment credential bound to that keypair (Document 01 §3).
- Enrollment credential is used once to establish the worker's identity, then retired in favor of session credentials.

### 3.2 Steady-State Operation
- Agent runs as a background service, establishes mTLS + session-token authenticated channel to the Control Plane.
- Continuously reports: health, resource availability, cached environment inventory, current job status (idle/running).
- Session tokens auto-refresh on a rolling basis; agent never operates on an expired token.

### 3.3 Job Execution
- Receives job assignment from Scheduler (via Control Plane).
- Resolves environment (cache hit or pull from Environment Registry, hash-verified either way — Document 01 §6).
- Creates isolated sandbox (Document 11), mounts environment read-only + fresh scratch space.
- Executes, captures framework-native checkpoints where applicable (Document 01 §5.1), reports progress.
- On completion: stage-1 output validation (local sanity check), upload result, wipe scratch space, destroy sandbox, return to idle.

### 3.4 Resource Throttling
- Agent continuously monitors host resource usage independent of job activity.
- Throttles or pauses job execution when the provider's own usage spikes (v1: simple threshold-based throttling; sophistication is a scale parameter for later).
- Never allows a job to cause the provider's machine to become unresponsive — this is a hard requirement, not a tunable default.

### 3.5 Failure & Recovery
- If the agent crashes mid-job, the sandbox and its scratch space must be recoverable/cleanable on next agent start — no orphaned processes or leftover data surviving a crash.
- If the agent loses connectivity to the Control Plane, in-progress jobs continue locally where possible but the agent must not accept new job assignments until reconnected and re-authenticated.
- Job reassignment (Document 01 §5) is triggered by the Control Plane's detection of a worker going silent past a health-check timeout.

### 3.6 Decommission / Uninstall
- Clean uninstall removes all cached environment layers, scratch space, credentials, and the agent binary itself.
- Enrollment identity is revoked on the Control Plane side upon explicit uninstall signal (where the agent can report it) or after a prolonged silence timeout.

## Senior Review
**Risks:** Crash recovery (3.5) is easy to under-test in a Pilot where the agent rarely crashes — should be deliberately fault-injected during testing rather than assumed correct.
**Performance Notes:** Environment caching (3.3) is what makes repeat jobs fast — this document's execution flow depends entirely on Document 01 §6's cache design being correct.
