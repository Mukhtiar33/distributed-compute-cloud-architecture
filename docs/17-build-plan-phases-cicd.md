# Document 17 — Build Plan: Phased Delivery, Testing & CI/CD

## Objective & Scope
Translate Documents 00–16 into an actual build sequence you can execute solo, phase by phase, without violating the core-architecture rule (Document 00 §1): every phase ships a *smaller version of the real thing*, never a fake/shortcut version that gets rebuilt later.

## Governing Rules for This Plan
1. **Every phase is independently testable** — both manually (your "command center," i.e. terminal/scripts you run yourself) and via automated GitHub Actions CI. No phase is "done" until both pass.
2. **No phase weakens Documents 01/11/12's security core** to move faster. Where a phase is smaller than production, it's smaller in *breadth* (fewer job types, smaller catalog), never in *rigor* (isolation, auth, validation stay full-strength from Phase 1 onward).
3. **Local-first, VPS-later, zero-rework migration.** Every phase runs on your laptop acting as the Control Plane host. Moving to a real VPS at the end is a deployment change (new host, same code, same config shape) — if any phase requires code changes to "become VPS-ready," that phase's design was wrong and needs fixing before you proceed, not after.
4. **FYP alignment:** each phase below is scoped so it can double as a chapter/milestone of your research (isolated, demonstrable, measurable) — noted per phase.

---

## Phase 0 — Foundations & Repo Skeleton

**Objective:** Get the skeleton every later phase builds on: repo structure, CI scaffolding, local dev environment, and the website shell (no real functionality yet).

**Deliverables:**
- Monorepo (or clearly separated repos) for: `control-plane/`, `worker-agent/`, `website/`, `docs/` (this series).
- Local dev environment reproducible via a single setup script (documents exact language/runtime versions used — Document 10's determinism principle applies to your own dev environment too).
- Website shell: static pages only — landing page, "Download Worker Agent" placeholder button (not yet wired to a real binary).
- GitHub repo with branch protection on `main`, PR-based workflow (even solo — this habit matters for FYP documentation and for when collaborators join).

**Exit Criteria:**
- Fresh clone + setup script → all three components build/run locally without manual fixing.
- Website shell loads locally and (optionally) is deployed to a free static host for early visibility.

**Manual Test (command center):**
- Clone repo fresh in a clean directory, run setup script, confirm each component starts without errors.

**GitHub Actions CI:**
- Lint + build check on every PR for each component (no tests yet — nothing to test).
- Fails the PR if any component fails to build.

**FYP Note:** This phase establishes your reproducibility baseline — worth documenting setup time and steps as a research artifact (reproducibility is a real HPC/distributed-systems research concern, ties naturally to your FYP).

---

## Phase 1 — Worker Agent MVP: Enrollment + Heartbeat (No Job Execution Yet)

**Objective:** Prove the Worker Agent can be installed, enroll itself with a (currently minimal) Control Plane, and maintain an authenticated heartbeat — the full auth model from Document 01 §3, at minimum scope.

**Deliverables:**
- Worker Agent binary: on first run, generates a local keypair, registers with the Control Plane, receives an enrollment credential, then a session credential.
- mTLS-secured heartbeat channel: agent reports "alive" every N seconds.
- Session token auto-refresh, working revocation (Control Plane can revoke a worker; agent's next heartbeat fails cleanly).
- Control Plane MVP: accepts enrollment, issues credentials, tracks connected workers, exposes a way to revoke one (even just a CLI command at this stage — no UI needed yet).

**Exit Criteria:**
- You can run 2+ worker agent instances (different terminals/VMs on your laptop) and see both enrolled and heartbeating in the Control Plane's state.
- Revoking one worker's session causes its next heartbeat to be rejected, and it must re-enroll to resume — proves the full two-tier model, not just a stub.

**Manual Test (command center):**
- Start Control Plane locally.
- Start 2 worker agents, confirm both show "enrolled + heartbeating."
- Kill and restart one agent, confirm it reconnects using stored credentials (not re-enrolling unnecessarily).
- Manually revoke a worker, confirm it's forced to re-enroll.
- Attempt to connect a worker agent with a tampered/expired token, confirm rejection.

**GitHub Actions CI:**
- Unit tests: credential generation, token expiry logic, mTLS handshake (can be tested against a local test CA, no real network needed).
- Integration test job: spins up Control Plane + 2 worker agent instances as CI job containers, runs the enrollment/heartbeat/revocation flow as an automated script, asserts expected state transitions.
- Fails the PR if enrollment, heartbeat, or revocation flow breaks.

**FYP Note:** This phase is a legitimate standalone research demonstration — "secure ephemeral node enrollment in a Zero Trust distributed system" — worth writing up even before job execution exists.

---

## Phase 2 — Job Intake & Manifest Validation (Control Plane, No Execution Yet)

**Objective:** Control Plane can accept a zip+manifest submission, validate the manifest schema (Document 10 §3), and reject malformed submissions — no scanning, no scheduling, no execution yet.

**Deliverables:**
- Consumer Interface MVP: accepts a zip upload via CLI/simple HTTP endpoint (no polished UI needed yet).
- Manifest parser + validator against the v1 schema.
- Job record created in Control Plane state (status: `intake_validated` or `rejected`), no further processing.

**Exit Criteria:**
- Valid manifest+zip → job accepted, status recorded.
- Missing/malformed manifest fields → rejected with a clear, specific error (not a generic failure).

**Manual Test:**
- Submit a well-formed manifest+zip, confirm acceptance.
- Submit manifests missing each required field one at a time, confirm each produces a specific rejection reason.

**GitHub Actions CI:**
- Unit tests covering the manifest schema: valid cases, each individual missing-field case, malformed-type cases.
- Fails PR on any schema regression.

---

## Phase 3 — Static Scanning Pipeline

**Objective:** Implement stage 1 of Document 01 §2 — static scanning of submitted job payloads before anything reaches a worker.

**Deliverables:**
- Dependency/package vulnerability scan integrated (existing open-source scanners are appropriate here — this is a good place to integrate a tool rather than build one from scratch).
- Static code pattern checks (known-dangerous patterns, obfuscation heuristics).
- Job status extended: `intake_validated → scanning → scan_passed | scan_rejected`.

**Exit Criteria:**
- A submission with a known-vulnerable dependency is flagged and blocked.
- A clean submission passes through to `scan_passed`.

**Manual Test:**
- Submit a job with an intentionally vulnerable dependency (test fixture), confirm rejection with the correct reason.
- Submit a clean job, confirm it passes.

**GitHub Actions CI:**
- Scanner integration tests using fixture payloads (one known-bad, one known-good) checked into the repo as test data.
- Fails PR if either fixture produces the wrong outcome (false negative or false positive on your own test set).

---

## Phase 4 — Sandbox Execution (Single-Worker Jobs, Full Isolation)

**Objective:** The core milestone — a worker agent can actually execute a job inside a properly isolated sandbox (Document 11), end-to-end, for the single-worker job path only (no parallel jobs yet, no dynamic detonation yet — those are separate phases).

**Deliverables:**
- Worker Agent: sandbox creation using the container-baseline mechanism (Document 11 §3) — process/filesystem/network/memory isolation, read-only environment mount + fresh scratch space (Document 11 §5).
- Environment resolution: for this phase, a **single hardcoded environment** is acceptable (defer full registry to Phase 5) — but it must still be mounted read-only and the execution must still be fully sandboxed. This is a breadth reduction, not a rigor reduction (Rule 2 above).
- Job execution: Control Plane assigns job to one worker (no ranking logic needed yet — just "any available worker" is fine for this phase), worker executes, uploads raw result (validation comes in Phase 6).
- Network policy default-deny inside the sandbox, per Document 11 §4.

**Exit Criteria:**
- A real job (e.g., a simple script from your chosen v1 environment) runs inside the sandbox on a worker and produces correct output.
- Sandbox cannot access the host filesystem outside its scratch+environment mounts (verified, not assumed).
- Sandbox has no network access unless explicitly required by the job (verify default-deny works).

**Manual Test (this is your most important manual security test — do it deliberately):**
- Run a job that attempts to read a file outside its scratch space — confirm it fails.
- Run a job that attempts outbound network access without an allowlist entry — confirm it's blocked.
- Run a job normally — confirm correct output and that scratch space is gone after completion (`ls` the expected scratch path post-job, confirm it's empty/destroyed).

**GitHub Actions CI:**
- Automated isolation test suite: the three manual tests above, scripted and run in CI on every PR touching the sandbox code — this is the single most important CI suite in the whole project and should block merges aggressively.
- Basic execution correctness test: known input → known expected output.

**FYP Note:** This phase alone (isolation verification methodology + results) is strong FYP material — a documented, reproducible, testable isolation guarantee is exactly the kind of contribution reviewers want to see evidence for, not just claims of.

---

## Phase 5 — Environment Registry & Caching

**Objective:** Replace Phase 4's hardcoded environment with the real mechanism from Document 10 — immutable, content-addressed, hash-verified, cached layers.

**Deliverables:**
- Environment Registry service (can live alongside the Control Plane initially).
- Worker Agent: cache check → hit (mount directly) or miss (pull, verify hash, cache, mount).
- Hash verification on every mount, not just download (Document 10 §2).
- At least 2 distinct environments registered, to prove the mechanism generalizes, not just works for one hardcoded case.

**Exit Criteria:**
- First job using a given environment: measurable pull+cache time.
- Second job using the same environment on the same worker: measurably faster (near-zero provisioning time) — this is the core performance claim from Document 04's success criteria and should be measured, not assumed.
- Tampering with a cached layer on disk (simulate manually) causes the next mount to fail hash verification, not silently execute a corrupted environment.

**Manual Test:**
- Run job A (environment 1) twice on the same worker, record provisioning time both times, confirm the second is materially faster.
- Manually corrupt a cached layer file, attempt to run a job requiring it, confirm hash verification catches it and the job doesn't execute against a tampered environment.

**GitHub Actions CI:**
- Cache hit/miss logic unit tests.
- Hash verification tampering test (automated version of the manual test above) — this is a security-critical path and belongs in CI, not just manual testing.

---

## Phase 6 — Three-Stage Output Validation

**Objective:** Implement Document 01 §2's full output validation pipeline (worker-local → Control Plane → pre-delivery).

**Deliverables:**
- Worker-local sanity check before upload.
- Control Plane authoritative validation against `expected_output` from the manifest.
- Pre-delivery integrity check at result assembly (trivial for single-worker jobs at this phase, becomes meaningful in Phase 8 with parallel jobs).
- Result delivered back to the consumer as a zip.

**Exit Criteria:**
- A job producing correct output passes all three stages and is delivered.
- A job producing output that doesn't match `expected_output` is caught at stage 2, not silently delivered.

**Manual Test:**
- Submit a job, confirm correct end-to-end delivery.
- Submit a job with a manifest whose `expected_output` deliberately doesn't match what the job actually produces, confirm stage-2 rejection with a clear reason.

**GitHub Actions CI:**
- Automated pass/fail fixture tests for each validation stage independently.

---

## Phase 7 — Dynamic Detonation Scanning

**Objective:** Add stage 2 of Document 01 §2 — sandboxed behavioral analysis before a job is trusted with a full worker slot.

**Deliverables:**
- Pre-execution detonation sandbox: heavily constrained (network-denied/throttled, syscalls/filesystem writes logged).
- Anomaly comparison against expected behavior for the job's declared type.
- Job status extended: `scan_passed → detonating → detonation_passed | detonation_flagged`.

**Exit Criteria:**
- A job with genuinely anomalous behavior (test fixture simulating unexpected network scanning, e.g.) is flagged and blocked from reaching a real worker.
- A normal job passes through without unnecessary delay.

**Manual Test:**
- Submit a fixture job designed to behave anomalously (e.g., attempts wide network access despite a "data preprocessing" declared type), confirm it's flagged.
- Submit a normal job, confirm it passes without false-positive flagging.

**GitHub Actions CI:**
- Fixture-based detonation tests (known-anomalous, known-normal), same discipline as Phase 3.

---

## Phase 8 — Independent-Parallel Jobs, Batch Grouping & Reassignment

**Objective:** Implement Document 01 §5 / Document 09's parallel job path: Control-Plane-grouped batches, per-batch status tracking, and failure-triggered reassignment — the most distributed-systems-heavy phase.

**Deliverables:**
- Manifest support for `job_type: parallel` with an input list.
- Control Plane batch grouping (simple round-robin/capacity-based, per Document 09 §4).
- Per-batch status model (`pending/running/succeeded/failed/reassigned`).
- Health-check-timeout-triggered reassignment.
- Idempotency handling for late/duplicate batch results (Document 09 §5).
- Full result assembly with stage-3 validation across all batches.

**Exit Criteria:**
- A parallel job with N inputs correctly splits into batches, runs across 2+ workers, and assembles a correct final result.
- Killing a worker mid-batch triggers correct reassignment without corrupting already-completed batches.
- A "zombie" worker reporting late success after reassignment does not double-count or corrupt the result (test this deliberately — it's the trickiest correctness case in the whole system).

**Manual Test:**
- Submit a parallel job across your 2 local workers, confirm correct split, execution, and assembly.
- Kill one worker mid-batch, confirm reassignment to the other worker, confirm final result is still correct.
- Simulate a zombie late-result (can be done by manually delaying one worker's response in a test harness), confirm it's discarded, not double-counted.

**GitHub Actions CI:**
- Automated multi-worker simulation (CI can spin up multiple worker agent containers): full split → execute → reassign-on-failure → assemble flow, scripted end-to-end.
- Dedicated idempotency/zombie-result test — this should be a named, permanent CI test given how easy this bug class is to reintroduce.

**FYP Note:** This phase is your strongest distributed-systems research chapter — fault-tolerant batch scheduling with reassignment and idempotency is a well-defined, measurable, citable contribution.

---

## Phase 9 — Website: Real Worker Agent Download + Consumer Upload UI

**Objective:** Replace Phase 0's placeholder website with the real functional front end.

**Deliverables:**
- Worker Agent download page, serving the real built binary (built by CI, per Section "Release Pipeline" below).
- Consumer upload UI: replaces Phase 2's CLI-only submission with a real form (upload zip+manifest, or a simple guided form that generates the manifest for less technical consumers).
- Basic account/identity handling for both roles (Document 05) — can be minimal (e.g., simple token-based accounts) but must issue **separate** consumer and provider identities even for a hybrid user (Document 05 §3), not a single shared one.

**Exit Criteria:**
- A non-technical user (good test: someone who hasn't seen the CLI version) can download the worker agent and run it, and separately submit a job, using only the website.

**Manual Test:**
- Full walkthrough as a first-time user, website only, no CLI fallback.
- Confirm hybrid-role identity separation (Document 05 §3) — register as both consumer and provider, confirm the platform doesn't preferentially route your own jobs to your own worker.

**GitHub Actions CI:**
- Website build/deploy pipeline (separate from the backend CI, per Section "Release Pipeline" below).
- Basic end-to-end UI test (can be a simple scripted flow, doesn't need to be exhaustive at this stage).

---

## Phase 10 — Monitoring/Logging Baseline

**Objective:** Implement Document 15's minimum structured logging and job-level tracing across all components, retroactively wired into every phase above.

**Deliverables:**
- Structured log schema (Document 15 §3) emitted at every job stage across all prior phases.
- Worker health signal logging.
- Environment cache hit-rate tracking (validates Phase 5's performance claim continuously, not just in a one-off manual test).

**Exit Criteria:**
- Given a job ID, you can reconstruct its full history (intake → scan → detonation → schedule → execute → validate → deliver) purely from logs, without needing to have watched it live.

**Manual Test:**
- Run a job, then — without looking at live output — reconstruct its full lifecycle purely from stored logs.

**GitHub Actions CI:**
- Log schema conformance tests (every stage emits the required fields, nothing silently skipped).

---

## Phase 11 — Full Local Integration Test ("Laptop as VPS")

**Objective:** Exactly what you described — run the Control Plane on your own laptop, 1–2 real separate worker machines, 1 real consumer, and validate the entire system against Documents 00–16 as a single end-to-end acceptance test.

**Deliverables:**
- A written test report mapping each Document 04 "Product Goals" success criterion to an observed pass/fail result.
- Any gaps found here get logged as issues, not silently patched without updating the corresponding architecture document — if the implementation needs to diverge from what's written in Docs 00–16, the docs get updated too (documentation rules from Document 00's governing rules: keep docs consistent with reality).

**Exit Criteria:** every Document 04 success criterion demonstrably passes on real, physically separate machines (not same-laptop simulated workers) — this is the real Pilot acceptance gate.

**Manual Test:** the full Document 04 goals table, executed literally, with results recorded.

**GitHub Actions CI:** at this stage, CI should already cover unit/integration/isolation/idempotency for every component — Phase 11's manual acceptance test is deliberately the one thing that stays manual, since it requires real separate physical machines CI can't fully replicate.

---

## Phase 12 — VPS Migration

**Objective:** Move the Control Plane from your laptop to a real VPS. Per Rule 3, this should require **zero application code changes** — only configuration (host address, TLS certificates for the real domain, environment variables).

**Deliverables:**
- VPS provisioned, Control Plane deployed via the same CI/CD release pipeline used for local testing (Section below).
- DNS/domain pointing at the VPS.
- Re-run Phase 11's full test report against the VPS-hosted Control Plane, with real remote workers (not on your local network).

**Exit Criteria:** identical results to Phase 11, on the real deployment target. Any discrepancy between local and VPS behavior indicates a hidden environment-coupling bug that should be fixed, not worked around.

---

## CI/CD Pipeline Design (applies across phases)

### GitHub Actions Structure
- `ci.yml` — runs on every PR: lint, unit tests, phase-relevant integration tests (as specified per phase above). Blocks merge on failure.
- `security-tests.yml` — runs the isolation (Phase 4), hash-tampering (Phase 5), and idempotency (Phase 8) test suites specifically, on every PR touching the relevant subsystem — these are flagged as their own workflow because they should never be allowed to become "just another test that got skipped in a rush."
- `release.yml` — triggered on merge to `main` or on tag: builds the Worker Agent binary (published to the website's download endpoint), builds/deploys the Control Plane container, deploys the website.

### Manual Testing ("Command Center") Discipline
For every phase, before merging, you personally run the manual test checklist listed under that phase, in your own terminal, against a locally running stack — this is not a replacement for CI, it's a check that CI's automated fixtures actually reflect real behavior (CI can pass while still missing something a human would immediately notice, especially in early phases before your fixture library is mature).

### Definition of Done (per phase, restated as a checklist)
- [ ] Deliverables implemented
- [ ] Exit criteria met
- [ ] Manual test checklist executed and passed by you personally
- [ ] GitHub Actions CI green on the relevant workflows
- [ ] Any deviation from Documents 00–16 reconciled by updating the relevant document, not left as silent drift

---

## Suggested Phase Order Summary

| Phase | Focus | Key Risk If Skipped/Rushed |
|---|---|---|
| 0 | Repo/CI skeleton | Reproducibility issues later |
| 1 | Worker enrollment + auth | Entire trust model unproven |
| 2 | Job intake/manifest | Garbage-in jobs reach later stages |
| 3 | Static scanning | Known-bad payloads reach workers |
| 4 | Sandboxed execution | **Core security guarantee unproven — highest-priority phase** |
| 5 | Environment caching | Core performance claim unproven |
| 6 | Output validation | Corrupted/malicious results reach consumers |
| 7 | Dynamic detonation | Novel-but-detectable attacks reach workers |
| 8 | Parallel jobs + reassignment | Data corruption under worker failure |
| 9 | Website | Unusable by anyone but you |
| 10 | Monitoring | Undebuggable in production |
| 11 | Local full integration | Unvalidated end-to-end claims |
| 12 | VPS migration | — (should be uneventful if everything above is correct) |

## Senior Review

**Potential Risks:** Phase 4 (sandbox execution) and Phase 8 (parallel jobs/reassignment) are both individually complex enough to become the project's actual bottleneck — budget disproportionate time for these two specifically rather than treating all phases as equal-sized.

**Security Notes:** The `security-tests.yml` workflow (isolation, hash-tampering, idempotency) is the single most important piece of automation in this plan — it's what prevents a future "quick fix" PR from silently reintroducing a sandbox escape or a double-counted batch result.

**Performance Notes:** Phase 5's cache hit-rate measurement should be captured as real numbers (not just "yes it's faster") — this becomes a legitimate FYP data point and also validates Document 10's core design bet.

**Scalability Notes:** Phase 12's "zero code change" migration is the practical test of Document 16's entire scaling philosophy — if it requires code changes, that's a signal some earlier phase leaked an environment-specific assumption (e.g., hardcoded localhost paths) that needs fixing at the source, not patched at migration time.

**Technical Debt:** None intentionally introduced by this plan — every phase reduces *breadth* (fewer environments, fewer job types) while keeping full *rigor* (Rule 2), consistent with Document 00's classification framework throughout.

**Suggested Future Improvements:** Once Phase 11 passes, consider whether any Document 04 success criteria need refinement based on what you actually observed — real testing often reveals a success criterion was either too strict or missed an edge case worth documenting.
