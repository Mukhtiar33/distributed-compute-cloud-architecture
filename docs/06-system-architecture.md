# Document 06 — Complete System Architecture

## Objective & Scope
Provide the end-to-end component map and data flow for v1, consistent with every decision made in Documents 00–05.

## 1. Components

| Component | Responsibility | Classification |
|---|---|---|
| Consumer Interface | Accepts zip+manifest submissions, returns result zips | CORE (v1 presentation layer) |
| Control Plane | Sole coordinator: auth, scanning, scheduling, environment resolution, output validation, metering | CORE |
| Scheduler (sub-component of Control Plane) | Matches jobs/batches to compatible workers | CORE contract, heuristic implementation in v1 |
| Environment Registry | Stores and serves immutable, hash-verified environment layers | CORE |
| Worker Agent | Runs on provider machines; sandbox lifecycle, environment caching, status reporting | CORE |
| Execution Sandbox | Per-job isolated environment (process/fs/network/memory) | CORE |
| Metering/Ledger store | Records resource consumption per job, per consumer/provider pair | CORE (Document 03 §3) |
| Monitoring/Logging | Structured logs, health signals, job-level tracing | CORE minimum, scale parameter depth |

## 2. High-Level Data Flow (Single-Worker Job)
1. Consumer uploads zip + manifest → Consumer Interface → Control Plane.
2. Control Plane runs static scan; if it fails, job is rejected immediately.
3. Job proceeds to sandboxed dynamic detonation; anomalies block promotion.
4. Scheduler selects a compatible, available worker based on manifest environment requirements.
5. Control Plane issues job to the Worker Agent over an authenticated (mTLS + session token) channel.
6. Worker Agent resolves environment: checks local cache (hash-verified) or pulls from Environment Registry.
7. Worker Agent creates an isolated sandbox, mounts the environment (read-only) + job scratch space, executes.
8. On completion: worker-local sanity check (validation stage 1) → upload result to Control Plane.
9. Control Plane validates against manifest's expected output + scanning (validation stage 2).
10. Sandbox is wiped; worker returns to idle; status reported.
11. Control Plane assembles final result (validation stage 3), records metering data, delivers zip to consumer.

## 3. High-Level Data Flow (Independent-Parallel Job)
Same as above through step 3, then:
4. Scheduler groups declared inputs into batches automatically (Document 01 §5, consumer never defines groupings).
5. Batches distributed across available compatible workers; each follows steps 5–9 above independently.
6. Control Plane tracks per-batch status (pending/running/succeeded/failed/reassigned); failed batches are reassigned without disturbing completed ones.
7. Once all batches succeed, Control Plane performs stage-3 validation across the full set (confirms none missing/corrupted) and assembles the final result.

## 4. Communication Topology
- Workers never communicate with each other — enforced at the network isolation layer (Document 01 §4), not just by convention.
- All worker ↔ Control Plane traffic is mTLS-authenticated with short-lived session tokens (Document 01 §3).
- Consumer Interface ↔ Control Plane is a separate trust boundary from Worker Agent ↔ Control Plane — a compromised consumer-facing endpoint must not grant any worker-level access.

## 5. Architecture Diagram (textual)
```
Consumer ---> Consumer Interface ---> Control Plane <---> Environment Registry
                                          |    ^
                                          v    |
                                     Scheduler |
                                          |    |
                                          v    |
                                  Worker Agent (per provider machine)
                                          |
                                          v
                                  Execution Sandbox (per job)
```

## Senior Review
**Risks:** The Control Plane is a single logical point of coordination in v1 (Document 00 §4) — this is an accepted single point of failure for the Pilot, explicitly deferred to horizontal scaling later. Must be documented as a known limitation, not discovered as a surprise.
**Scalability Notes:** Every arrow in the diagram above is designed to remain valid when the Control Plane becomes multiple instances — workers and consumers address "the Control Plane" as a logical endpoint, not a specific host, even in v1's single-instance implementation.
