# Document 01 — Core Scope Definition (v1 / Pilot)

## Status
Supersedes the preview classification in Document 00, Section 3, with concrete decisions. This is now the authoritative core-vs-deferred boundary for v1.

## Relationship to Document 00
The governing rule from Document 00 still holds: **trust boundaries do not shrink with scale.** Everything below is CORE, meaning it exists in full in v1, running on a handful of friends' laptops, with the same rigor it will have at production scale. Nothing here is a "good enough for now" shortcut.

---

## 1. Core Subsystems (v1)

### 1.1 Control Plane

| Capability | v1 Scope | Notes |
|---|---|---|
| Consumer workload vetting | Static scanning + sandboxed dynamic detonation before full execution | See Section 2 |
| Provider-side data validation | Output/result scanning before it reaches the consumer or the Control Plane's durable storage | Symmetric to consumer-side vetting — a compromised or misbehaving worker's output is untrusted data too |
| Worker authentication | Two-tier credential model (enrollment + session) | See Section 3 |
| Cross-job isolation | Strict per-job process/filesystem/memory/network isolation, even when one worker runs jobs for two different consumers sequentially or concurrently | See Section 4 |
| Scheduling | Supports **both** single-worker (non-parallel) jobs and sharded (data-parallel) jobs from v1 | See Section 5 |
| Environment provisioning | Immutable, cached, content-addressed environment layers — not full reinstall per job | See Section 6 |
| Manifest requirement | Every submitted job includes a minimal declarative manifest | See Section 7 |

Scheduling intelligence itself (heuristic in v1, ML-driven later) is a **scale parameter**, not core — but the four capabilities above are the *contract* the Control Plane must fulfill regardless of how smart the decision-making behind them is.

### 1.2 Worker Agent
- Single downloadable binary, distributed from the platform website.
- Runs as a background service on the provider's machine.
- Responsibilities (all core, all v1):
  - Continuous status/health reporting to the Control Plane.
  - Sandbox lifecycle management: create, populate (from cached environment + job payload), execute, tear down, securely wipe.
  - Environment layer download/cache management (working with the Control Plane's environment registry).
  - Sole communication channel between the provider machine and the platform — workers never talk to each other, and nothing on the provider machine talks to the Control Plane except through the agent.
  - Enforcing resource throttling so the provider's normal machine use isn't disrupted (the *precision* of throttling heuristics is a scale parameter; the *existence* of throttling is not).

### 1.3 Consumer Interface
- v1 interaction model: upload a zip (project + manifest) → platform returns a zip (results).
- This is a deliberate simplification of the long-term "invisible infrastructure, natural-language job description" vision from Document 00 — classified as a **FUTURE FEATURE** wrapper around a CORE execution pipeline. The zip-in/zip-out interface is just the v1 presentation layer; the manifest-driven execution contract underneath it is the same contract the future natural-language interface will eventually compile down to. We are not building a different backend later — we are building a different front door.

---

## 2. Malicious Workload Vetting

**Decision: static scanning + sandboxed dynamic detonation before full execution.**

Two-stage pipeline, both mandatory:

1. **Static stage** (fast, cheap, runs on every submission):
   - Dependency/package scanning against known-vulnerability databases.
   - Static code analysis for known-dangerous patterns (obfuscated payloads, suspicious syscalls referenced in source, known malware signatures).
   - Manifest/entrypoint sanity checks (does the declared entrypoint exist, does the declared environment match what the code imports).
   - Fails fast — a job that trips static scanning never reaches a worker.

2. **Dynamic detonation stage** (sandboxed, before the job is trusted with a real worker slot):
   - Job runs first in a tightly constrained, heavily monitored detonation sandbox (network-denied or heavily throttled, filesystem writes captured, syscalls logged).
   - Behavior is compared against expected patterns for its declared job type (e.g., a "data preprocessing" job that suddenly attempts wide network scanning is anomalous).
   - Anomalies block promotion to a real provider worker and flag for review.

**Honest limitation, stated explicitly so it's never forgotten:** this pipeline catches known and behaviorally-anomalous threats. It does **not** guarantee detection of a novel, well-disguised attack. This is why isolation (Section 4) is a mandatory independent layer, not a fallback for when scanning "misses something" — the system must be safe even in the case where scanning passes a malicious job.

**Provider-side symmetry — three-stage output validation (defense-in-depth, not redundant repetition):** a worker's output/result is never trusted by default. Each stage catches a different failure mode:

1. **On the worker, pre-upload** — sandbox-local sanity check (expected output files exist, sizes look plausible) before anything leaves the provider's machine. Catches ordinary job/worker bugs early and cheaply.
2. **On receipt at the Control Plane** — authoritative validation against the manifest's `expected_output`, plus the scanning pipeline (this section). Catches a compromised or malicious worker attempting to use its result payload as an attack vector against the Control Plane.
3. **Before final delivery to the consumer** — integrity check at result-assembly time, especially for parallel jobs: confirms all batches are present and uncorrupted before the final zip is built. Catches assembly-layer bugs (a dropped or corrupted batch that individually passed stage 2 but is missing from the whole).

No stage substitutes for another — a job could pass stage 1 (worker's own bug-free reporting) while still being malicious (caught at stage 2), or pass stages 1–2 individually while the assembled whole is incomplete (caught at stage 3).

---

## 3. Worker Authentication — Two-Tier Model

**Decision: formalize enrollment credential and session credential as separate mechanisms.**

| Tier | Purpose | Lifetime | Mechanism |
|---|---|---|---|
| **Enrollment credential** | Proves "this worker agent was legitimately obtained from our platform," issued once when the agent is first installed/registered | Long-lived but single-use for its purpose — it enrolls the worker and is then retired, replaced by session credentials | Issued by the Control Plane at first registration; tied to the specific worker install (bound to a generated worker identity, ideally backed by a locally-generated keypair so the enrollment credential is never the sole secret in play) |
| **Session credential** | Authenticates the ongoing agent ↔ Control Plane channel for actual operation | Short-lived (minutes, not the originally proposed 24 hours), auto-refreshed by the agent while alive, immediately revocable server-side | Short-lived signed token (JWT or equivalent), issued after successful enrollment or re-authentication, refreshed on a rolling basis |
| **Transport identity** | Prevents any non-authenticated party from even opening a channel to the Control Plane, independent of the token layer | Per-connection | Mutual TLS — the worker agent presents a client certificate tied to its enrolled identity; this replaces the original "TCP kind of checking" idea with a standard, well-audited mechanism |

**Why this is better than a single 24-hour rotating code:**
- A single shared code that rotates daily has a large blast radius if intercepted or logged — anyone holding it for up to 24 hours can impersonate a worker.
- Splitting enrollment from session means a leaked *session* token expires in minutes and is trivially revocable, while the *enrollment* credential (rarer, higher-value) is used once and never travels over the wire again afterward.
- Mutual TLS means even a stolen session token is useless without the worker's private key, which never leaves the worker's machine.

**Compromise/rotation handling (must be specified before implementation, not deferred):**
- If a worker is flagged (failed scans, anomalous behavior, provider report), its session credentials are revoked immediately and re-enrollment is required — a deliberately higher-friction path than routine session refresh.
- Enrollment credential compromise (e.g., someone steals the install package's embedded secret) must not by itself grant lasting access — this is why enrollment is bound to a worker-generated keypair rather than being a bearer secret alone.

---

## 4. Cross-Job Isolation

**Decision: one worker may run jobs for two different consumers, but isolation is per-job, not per-worker.**

Required isolation boundaries per job, all mandatory in v1:
- **Process isolation** — separate process/container namespace per job.
- **Filesystem isolation** — each job sees only its own ephemeral scratch space plus read-only access to the shared, immutable environment cache (Section 6). No job can see another job's files, even from the same worker, even sequentially (scratch space is wiped between jobs, not merely "closed").
- **Network isolation** — a job's network access is scoped to only what its declared environment/manifest requires (which for most jobs should be nothing, or a narrow allowlist); Job A cannot reach Job B's traffic or the host's general network.
- **Memory isolation** — standard OS/container-level memory isolation, no shared memory segments between jobs.
- **Secrets isolation** — any credentials the sandbox itself needs (e.g., to pull an environment layer) are scoped to the agent, never exposed inside the job's execution context.

This directly operationalizes Document 00's sandbox philosophy — nothing new in principle, just precision on the "two consumers, one worker" case, which is the concrete scenario most likely to surface an isolation bug in testing.

---

## 5. Scheduling — Both Job Types Supported in v1 (Revised)

**Decision: support single-worker (non-parallel) and independent-input-parallel jobs from v1. Distributed multi-worker training with gradient synchronization is explicitly deferred.**

Revision rationale: an earlier draft of this document proposed "consumer-declared sharding," which quietly pushed a splitting decision onto the consumer — this violates the platform's core invisibility principle (Document 00) and was corrected after review. Two structurally different problems were being conflated under "sharding":

1. **Independent-input parallelism** — jobs whose units of work don't depend on each other (batch data processing, rendering, batch inference). The consumer supplies multiple inputs as part of their normal upload; the Control Plane decides batch boundaries automatically based on worker availability, with zero consumer involvement.
2. **Distributed model training with gradient synchronization** — a single continuous training process that would need to act as a parameter-server/aggregator across workers, handle mid-epoch worker dropout, and synchronize weight updates without corrupting the model. This is a hard distributed-systems problem independent of anything the consumer could declare, and is **not** in v1 scope.

| Job Type | v1 Scheduling Behavior | Result Assembly |
|---|---|---|
| **Single-worker** (includes all model training jobs in v1) | Entire job assigned to one compatible worker; no splitting. Matches the consumer's existing mental model exactly — "start it, wait for results," same as running locally | Worker's output zip is the result, after provider-side validation (Section 2, three-stage — see Section 2 revision) |
| **Independent-input-parallel** | Consumer's inputs (declared as a list in the manifest, not as "shards") are automatically grouped into batches by the Control Plane, distributed across available compatible workers | Control Plane gathers all batch results, validates each, assembles into a single result zip for the consumer |

**Consumer burden: zero in both cases.** The consumer never declares a split strategy, a checkpoint, or a shard boundary. For parallel jobs, the manifest lists *inputs*, not *shards* — the grouping decision belongs entirely to the Control Plane.

**Failure handling (must exist in v1, not deferred):** a batch that fails or times out on one worker must be reassignable to another compatible worker without corrupting already-completed batches. This requires batch-level idempotency and a status model (pending/running/succeeded/failed/reassigned) tracked by the Control Plane. For single-worker jobs (including training), a failed/dropped worker requires the whole job to be reassigned — this is a real UX cost of not yet having distributed training, and should be mitigated with periodic framework-level checkpoint capture (see Section 5.1) rather than losing all progress.

### 5.1 Mitigating single-worker job loss (v1, without full distributed training)
Even without gradient-sync distribution, v1 should still capture whatever native checkpointing the job's own framework produces (e.g., PyTorch's own periodic checkpoint files) into the job's scratch space, so that if a worker drops mid-job, reassignment can resume from the last framework-native checkpoint on a new worker rather than restarting from zero. This is a meaningfully smaller engineering lift than building distributed synchronization, and directly improves the "worker disappears mid-training" failure mode.

### 5.2 Deferred: Distributed Multi-Worker Training
Named explicitly so it isn't silently forgotten: true distributed training (gradient synchronization across multiple workers for a single training run) is a FUTURE capability, requiring its own design document once v1's single-worker and independent-parallel paths are proven. It will need to address: parameter-server vs. all-reduce topology choices, worker dropout mid-sync, and how this interacts with the platform's Zero Trust worker isolation (workers in a distributed training job need to exchange gradients — a new communication pattern that never existed in the "workers never talk to each other" model from Document 00, and needs careful design, likely still mediated through the Control Plane rather than direct worker-to-worker).

---

## 6. Environment Provisioning — Immutable Cached Layers

**Decision: content-addressed, immutable environment layers, cached on the worker, reused across jobs.**

- Environments (framework + dependency sets) are packaged as immutable, hash-verified layers, analogous to container image layers.
- A worker downloads a given environment layer **once**; subsequent jobs requiring the same environment reuse the cached, verified layer rather than re-downloading or reinstalling.
- **Explicit amendment to Document 00's sandbox philosophy:** "nothing persists unless explicitly required by platform policy" now has one precise, documented exception — the environment layer cache persists across jobs. It is not consumer data and not provider data; it is platform-distributed, hash-verified software, and its persistence is exactly the "explicitly required by platform policy" carve-out the original philosophy anticipated.
- What must still be fully wiped between every job, without exception: the job's own scratch space, any files it wrote, any in-memory state, any environment variables/secrets injected for that specific job.
- Cached layers are **read-only** to every job that mounts them — no job can modify the shared cache, which is what keeps the "environment cache persists" exception from becoming a cross-job contamination vector.
- Layer integrity: every cached layer is verified against a known hash before use, on every mount, not just at download time — protects against a compromised worker's local disk being used to inject a tampered environment.

---

## 7. Manifest Requirement

**Decision: every job submission requires a minimal declarative manifest, alongside or inside the zip.**

Minimum v1 manifest fields (to be formalized precisely in the Execution Environments document):
- `job_type`: `single` | `sharded`
- `environment`: declared framework/dependency identifier (keys into the environment registry)
- `entrypoint`: what to execute
- `shards` (if `job_type: sharded`): explicit shard definition, consumer-declared per Section 5
- `expected_output`: minimal shape description, used for provider-side output validation (Section 2)

Without this, the Control Plane has no basis for environment resolution, scheduling, or output validation — it's not optional metadata, it's the contract the rest of the pipeline is keyed off.

---

## 8. Updated Core vs. Deferred Table

| Item | Classification |
|---|---|
| Static + dynamic malicious workload scanning | CORE |
| Provider-side output validation | CORE |
| Two-tier auth (enrollment + session) + mTLS | CORE |
| Per-job isolation (process/fs/network/memory/secrets) | CORE |
| Single-worker job scheduling | CORE |
| Independent-input-parallel scheduling (Control-Plane-grouped batches, zero consumer burden) | CORE |
| Framework-native checkpoint capture for single-worker job resumption | CORE |
| Distributed multi-worker training (gradient sync across workers) | FUTURE (explicitly deferred, not designed yet — see §5.2) |
| Immutable cached environment layers | CORE |
| Environment catalog breadth (many frameworks) | SCALE PARAMETER |
| Job manifest schema | CORE (minimal v1 schema, extensible later) |
| Worker agent single-binary, background service | CORE |
| Zip-in/zip-out consumer interface | v1 presentation layer (FUTURE: natural-language front end, same backend contract) |
| ML-driven scheduling intelligence | SCALE PARAMETER (heuristic in v1, same interface later) |
| Credit economy, pricing, marketplace matching | DEFERRED (metering seam only, per Document 00 §3.6) |
| Multi-region, horizontal Control Plane scaling | DEFERRED (Document 00 §3.8) |

---

## Senior Review

**Potential Risks:** The sharded-job path (Section 5) is the newest and least-specified piece of this scope — "who defines shard boundaries" is flagged as an open question with a recommendation (consumer-declared), but it needs your sign-off before the Scheduler document formalizes it, since it affects the manifest schema everything else depends on.

**Security Notes:** The two-tier auth model plus mutual TLS is a meaningful strengthening over the original 24-hour code proposal, with no added user-facing complexity (the provider still just runs one agent) — the complexity is entirely on the platform side, which is where it belongs.

**Performance Notes:** The immutable environment cache is the single biggest performance lever in this design — it turns "reinstall everything every job" into "download once, reuse forever until the environment version changes." Cache invalidation strategy (what happens when an environment layer is updated) is not yet specified and should be in the Execution Environments document.

**Scalability Notes:** Nothing in this document requires rework to scale — the two-tier auth, per-job isolation, and layer caching all work identically at 5 workers or 5 million. The only thing that changes with scale is the Control Plane's own scheduling throughput, which is explicitly out of scope for this document (Document 00 §3.8).

**Technical Debt (intentionally deferred):** automatic shard inference, environment catalog breadth, ML scheduling model, credit economy/pricing, multi-region deployment.

**Resolved since first draft:** sharding model corrected from consumer-declared to Control-Plane-grouped independent-input batching (Section 5); training jobs run single-worker in v1 with framework-native checkpoint capture rather than distributed sync (Section 5.1–5.2); output validation formalized as three-stage defense-in-depth (Section 2).

**Open items requiring your decision before the next document:**
1. Any constraints on what "expected_output" validation can realistically check for arbitrary job types (this affects how strict provider-side output validation in Section 2 can be) — e.g. is structural/size/type checking sufficient for v1, or do you want content-aware checks for known output types (model checkpoint files, image outputs, etc.)?
