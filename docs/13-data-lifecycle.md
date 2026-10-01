# Document 13 — Data Lifecycle

## Objective & Scope
Trace every category of data through its full lifecycle — creation, storage, access, and destruction — across the platform.

## 1. Data Categories

| Category | Examples | Persists? |
|---|---|---|
| Consumer job payload | Uploaded zip contents | No — exists only for job duration, deleted after result delivery (retention window TBD, default short) |
| Job scratch space | Files written during execution | No — wiped every job (Document 11 §5) |
| Environment layers | Framework/dependency packages | Yes — immutable, hash-verified, explicit exception (Document 01 §6) |
| Metering/ledger records | Resource consumption per job | Yes — durable, is the economic foundation (Document 03) |
| Scan/validation logs | Static/dynamic scan results, output validation outcomes | Yes, retention policy TBD (Document 12 §10) |
| Worker identity records | Enrollment keypair binding, session history | Yes — needed for revocation/re-enrollment (Document 01 §3) |
| Monitoring/health data | Worker status reports, job traces | Yes, scale-dependent depth (Document 15) |

## 2. Consumer Job Payload Lifecycle
1. Uploaded to Consumer Interface.
2. Passed to Control Plane, scanned (Document 01 §2).
3. Distributed to worker(s) as needed for execution.
4. Deleted from Control Plane storage after result delivery — a specific retention window (e.g., short grace period for re-download) should be defined before implementation, not left implicit.
5. Never stored longer than operationally necessary; no indefinite retention by default.

## 3. Environment Layer Lifecycle
Covered fully in Document 10 §2, §6 — creation (published to registry), distribution (cached on workers), invalidation (versioned, not overwritten), garbage collection (local LRU eviction on workers, always re-fetchable).

## 4. Metering Data Lifecycle
Created per job completion (Document 03 §3), stored durably, never deleted (it is the historical record the future economy depends on). Should be designed to minimize sensitive content — records resource consumption and identifiers, not job payload content (Document 12 §6).

## 5. Cross-Border / Regional Data Considerations (flagged, not resolved)
v1 is single-region (Document 00 §4), so this is not an immediate concern, but the data lifecycle design should not assume payload data can freely cross regions once multi-region deployment (Document 16) happens — flagged as an open question for that document rather than answered here.

## 6. Deletion Guarantees
"Deleted" must mean actually removed, not merely dereferenced — particularly for consumer job payloads and job scratch space, given the platform's core promise that no consumer data persists on provider machines or in Control Plane storage beyond its necessary window (Document 02 §2).

## Senior Review
**Privacy/Security Notes:** The retention window for consumer job payloads (Section 2, step 4) is an explicit open item — needs a concrete default (e.g., 24–72 hours post-delivery) before implementation, not left as "delete eventually."
**Technical Debt:** Cross-border data handling (Section 5) is correctly deferred given v1's single-region scope, but should be revisited early in Document 16's design rather than as an afterthought once multi-region work begins.
