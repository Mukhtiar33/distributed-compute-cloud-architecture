# Document 00 — Program Roadmap & MVP Scope Definition

## Status
Foundational reference. Every other document in this series must remain consistent with the boundaries defined here.

## Objective
Define how the platform scales from a **Pilot** (friends' laptops, single operator, a handful of workloads) to a **Production SaaS** (anonymous global providers, millions of users) *without ever re-architecting the trust, security, or orchestration model*. This document draws the line between what is **core and immutable from day one** and what is a **scale-dependent parameter** we're permitted to simplify, defer, or hardcode for the Pilot.

The failure mode we are explicitly designing against: building a v1 that trusts providers, skips sandbox isolation, or hand-schedules jobs "because it's just friends," and then discovering at v2 that the entire execution model has to be rebuilt because untrusted-provider assumptions were never load-bearing in the first place. That rebuild is more expensive than building it right the first time, even at tiny scale.

---

## 1. The Governing Rule

> **Trust boundaries do not shrink with scale. Only throughput, geography, and population do.**

Concretely: a workload running on your friend's laptop in the Pilot must be sandboxed, verified, and cleaned up with *exactly* the same rigor as a workload running on an anonymous stranger's machine in Production. The only thing allowed to differ is *how many* jobs are scheduled per second, *how many* regions are supported, and *how sophisticated* the ML behind the scheduler is.

Why this matters: security and isolation guarantees are not additive features you bolt on later — they are structural. A sandbox architecture designed under the assumption "I trust these 10 people" will leak that assumption into IPC design, filesystem layout, credential handling, and cleanup logic in ways that are difficult to retrofit. Building it Zero Trust from day one costs marginally more now and saves a full subsystem rewrite later.

---

## 2. Classification Framework

Every design decision in every subsystem gets tagged as one of three types:

| Tag | Definition | Rule |
|---|---|---|
| **CORE** | Structural to the trust model, security model, or orchestration contract. Changing this later requires touching every workload that ever ran. | Must be fully implemented in the Pilot, even if usage is small. Never mocked, stubbed, or trusted-shortcut. |
| **SCALE PARAMETER** | Governs throughput, capacity, or geographic reach. Changing this later requires infrastructure work, not architectural rework. | Can be hardcoded, single-instance, or manually operated in the Pilot. Must have a clear, documented upgrade path. |
| **FUTURE FEATURE** | Adds new capability or new user-facing surface area. Absence does not compromise anything already running. | Deferred entirely. Not designed in detail yet — only named, so nothing core accidentally forecloses it. |

Every subsystem document in this series will open with a table classifying its major components against this framework.

---

## 3. Classification by Subsystem (Preview)

This is a preview; each item gets full treatment in its own document.

### 3.1 Security, Sandboxing, Trust Model → **100% CORE**
- Workload authentication, verification, scanning before execution
- Filesystem/process/memory/network isolation per job
- Zero Trust communication between every pair of components
- No consumer data persists on provider machines after execution
- No provider data becomes accessible to workloads
- Secrets never touch the sandbox in plaintext outside a scoped injection mechanism

**None of this is negotiable in the Pilot.** Running on a friend's laptop does not mean the friend's laptop is trusted — it means the *blast radius of a compromise* is small, not that the compromise is acceptable. We build the isolation boundary now while the cost of getting it wrong (in bugs found, in design iteration) is cheapest.

### 3.2 AI-Native Control Plane → **CORE contract, SCALE PARAMETER sophistication**
- The *contract* is core: the Control Plane is the sole coordinator, workers never talk to each other, all scheduling decisions flow through one logical authority.
- The *intelligence* is a scale parameter: a Pilot-stage scheduler can use simple heuristics (rule-based matching on declared hardware, historical success rate) standing behind the *same interface* a future ML-driven predictor will use. We are not required to have a trained model on day one — we are required to have the seam where one plugs in.

### 3.3 Worker Agent → **100% CORE**
- Single lightweight install, background service, no manual dependency install by the provider — this is core to the product philosophy itself, not just security, and must hold at N=5 or N=5,000,000.
- Environment provisioning automation, resource throttling to avoid disrupting the provider: core behavior, though the *aggressiveness/precision* of throttling heuristics is a scale parameter that gets more refined over time.

### 3.4 Scheduler & Orchestration → **CORE contract, SCALE PARAMETER implementation**
- Job → environment → compatible worker matching, checkpointing, verification, cleanup: core lifecycle, must exist end-to-end in the Pilot.
- Global bin-packing optimization across thousands of concurrent jobs, predictive pre-warming, multi-region scheduling: scale parameters, deferred.

### 3.5 Execution Environments / Environment Registry → **CORE contract, SCALE PARAMETER breadth**
- The mechanism (declarative environment spec → reproducible sandboxed execution) is core.
- The *catalog size* (number of supported frameworks/images) is a scale parameter — Pilot can support a narrow set (e.g. one or two ML frameworks) as long as the registry mechanism itself is general, not hardcoded per-framework.

### 3.6 Credit Economy / Marketplace → **FUTURE-LEANING, with a CORE seam**
- Full pricing, matching markets, abuse/fraud economics: not needed to prove the core execution model, deferred in depth.
- However: the *accounting seam* (every job's resource consumption is metered and attributable to a consumer/provider pair) is CORE — because retrofitting metering into an execution path that wasn't built to record it is another expensive rebuild. We meter from day one even if we don't yet price.

### 3.7 Monitoring & Observability → **CORE minimum, SCALE PARAMETER depth**
- Structured logging, job-level tracing, and basic health signals from every component are CORE — without them, debugging the security/sandbox model itself is impossible.
- Full distributed tracing, dashboards, alerting pipelines, SLO tracking: scale parameters, built out as operator count grows.

### 3.8 Scaling Infrastructure (horizontal scaling, multi-region, CDN, load balancing) → **100% FUTURE / SCALE PARAMETER**
- Single-instance Control Plane, single region, no CDN, no load balancer: acceptable for Pilot.
- Must be designed so scaling out is an infrastructure change (add instances, add regions) rather than a protocol change (workers/consumers should not need to know how many Control Plane instances exist).

---

## 4. What "Pilot" Concretely Means

- **Population:** you + a small number of known participants (friends' laptops), manually onboarded.
- **Geography:** single region, no redundancy requirement.
- **Control Plane:** single logical instance (may be a single process/host).
- **Scheduler intelligence:** heuristic-based, not ML-trained — but sitting behind the same decision interface the future ML scheduler will use.
- **Environment catalog:** narrow, intentionally limited to the workload types you want to prove first.
- **Credit economy:** metering on, pricing/marketplace matching off or trivial (flat rate / manual).
- **Security posture:** full Zero Trust, full sandboxing, full lifecycle cleanup — **unchanged from Production.**

## 5. What Graduates the Pilot to v2

A component graduates when its **SCALE PARAMETER** ceiling is reached — not on a calendar date. Examples of graduation triggers we'll define precisely in each subsystem doc:
- Control Plane single-instance throughput saturates → introduce horizontal scaling of the Control Plane (requires the Control Plane's internal state to already be externalized, which we'll specify as CORE in its own document even though scaling is deferred).
- Manual onboarding doesn't scale past N known providers → introduce self-serve provider registration, which is where identity/reputation systems become load-bearing.
- Heuristic scheduler starts making visibly bad matches at higher job diversity → swap in the trained model behind the existing scheduling interface.

---

## 6. Open Questions (require resolution before implementation, not before design)

1. Where does job metering data live in the Pilot — local to the Control Plane instance, or already in an externally durable store? (Affects how painful the v2 externalization is.)
2. What is the minimal viable "environment spec" schema for the Pilot's narrow framework catalog, such that it doesn't need to be redesigned when the catalog grows?
3. What identity mechanism do Pilot providers/consumers use (you're manually onboarding friends) that still exercises the *real* authentication path rather than a bypass?
4. How much of the AI-native scheduler's future feature set (predictive availability, reliability scoring) needs its data-collection hooks built now, even if the model itself doesn't exist yet?

These will be resolved in their respective subsystem documents, not here — this document only establishes that they must be resolved deliberately, not defaulted away.

---

## 7. Document Series (Planned)

1. ~~Program Roadmap & MVP Scope Definition~~ *(this document)*
2. Project Vision & Philosophy
3. Business Model
4. Product Goals
5. User Roles (Provider, Consumer, Hybrid)
6. Complete System Architecture
7. AI-Native Control Plane
8. Worker Agent Philosophy & Lifecycle
9. Scheduler & Intelligent Orchestration
10. Execution Environments
11. Sandboxing & Isolation
12. Security, Privacy & Trust Model
13. Data Lifecycle
14. Credit Economy
15. Monitoring & Observability
16. Scaling Philosophy

Each document will open with a **Core vs. Scale Parameter vs. Future Feature** classification table per Section 2 of this document, so the whole series stays internally consistent as it grows.

---

## Senior Review

**Potential Risks:** The biggest risk to this plan isn't technical, it's discipline — the temptation to shortcut CORE items "just for the pilot" is exactly how trust-boundary debt gets introduced. Every future document needs to be checked against this one before anything gets marked "good enough for now."

**Security Notes:** Classifying sandboxing/isolation as 100% CORE from day one is the single highest-leverage decision in this document — it's the one category where retrofitting is not just expensive but potentially requires re-auditing every workload that ever ran under the weaker model.

**Scalability Notes:** The single largest unresolved scaling question is Control Plane state externalization (Open Question 1) — if we get this wrong in the Pilot, horizontal scaling of the Control Plane later becomes a rewrite rather than an infrastructure change.

**Technical Debt (intentionally deferred, documented so it isn't forgotten):** marketplace pricing/matching, multi-framework environment catalog breadth, multi-region deployment, ML-trained scheduler model, self-serve provider onboarding/identity.

**Suggested Next Step:** Document 2 (Vision & Philosophy) or Document 12 (Security/Trust Model) — I'd lean toward doing Security & Trust Model early, before Architecture, since it's the one subsystem that constrains every other document's design space rather than the other way around. Your call.
