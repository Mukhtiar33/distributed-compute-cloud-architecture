# Document 04 — Product Goals

## Objective & Scope
Translate the vision (Document 02) into concrete, testable goals for v1, with clear success criteria.

## 1. v1 (Pilot) Goals

| Goal | Success Criterion |
|---|---|
| Prove the zip-in/zip-out execution loop | A consumer can submit a project zip + manifest and receive a correct result zip, for both single-worker and independent-parallel job types (Document 01 §5) |
| Prove full-rigor security/isolation at small scale | Every job runs through two-tier auth, mTLS, three-stage output validation, and per-job isolation — with zero shortcuts relative to the Production design |
| Prove environment caching solves the reinstall problem | Second and subsequent jobs using a previously-seen environment show materially faster provisioning than the first (no full reinstall) |
| Prove the Worker Agent doesn't disrupt providers | Providers can continue normal machine use while jobs run, with throttling behavior observable and acceptable |
| Prove failure recovery works | A worker dropping mid-job (single-worker or a batch of a parallel job) results in correct reassignment without data corruption or silent job loss |

## 2. Explicit Non-Goals for v1
- Marketplace pricing, billing, payouts (Document 03).
- Multi-region deployment, horizontal Control Plane scaling (Document 00 §3.8).
- Distributed multi-worker training with gradient sync (Document 01 §5.2).
- Self-serve, anonymous provider onboarding (Pilot providers are manually enrolled, known individuals).
- Natural-language job submission (zip interface only, per Document 01 §1.3).

## 3. Qualitative Product Goals
- The consumer experience should feel indistinguishable from running the job locally, minus the wait for available compute — no new concepts (shards, environments, workers) exposed to them.
- The provider experience should feel like installing a lightweight background utility, not administering a server.

## Senior Review
**Risks:** The temptation to declare "v1 done" before the failure-recovery goal is actually exercised (workers rarely drop in a small friendly Pilot) — this should be deliberately tested, not assumed to work because it was designed correctly.
**Suggested Future Improvement:** Add explicit acceptance tests per goal before implementation begins, so "success criterion" isn't just descriptive but executable.
