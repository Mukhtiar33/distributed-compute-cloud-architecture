# Document 11 — Sandboxing & Isolation

## Objective & Scope
Specify the mechanism behind Document 01 §4's isolation requirements — this document answers "how," where Document 01 established "what" and "why."

## 1. Core Classification
Everything in this document is CORE — isolation mechanism is the single highest-leverage security decision in the platform (Document 00 §1) and has no scale-dependent relaxation.

## 2. Isolation Boundaries Required (recap from Document 01 §4)
Process, filesystem, network, memory, and secrets isolation — per job, not per worker, enforced even when one worker sequentially or concurrently runs jobs for two different consumers.

## 3. Mechanism Options (comparative)

| Approach | Isolation Strength | Overhead | v1 Fit |
|---|---|---|---|
| OS process isolation only (no containers) | Weak — shared kernel, shared filesystem namespace by default | Lowest | Not sufficient — rejected |
| Containers (namespaces + cgroups) | Moderate-strong — separate namespaces, resource limits, but shared kernel | Low | Acceptable baseline for v1, given consumer-provided code is scanned pre-execution (Document 01 §2) as a first line of defense |
| MicroVMs (e.g., lightweight VM-based sandboxing) | Strong — separate kernel per job, much smaller attack surface for kernel-level escapes | Moderate (higher than containers, much lower than full VMs) | Preferred target once justified by risk profile / provider hardware constraints |
| Full VMs | Strongest | High (slow to spin up/down, resource-heavy) | Not a good fit — conflicts with the "fast, ephemeral, per-job" execution model |

**v1 recommendation: containers as the baseline, with microVM-level isolation as a near-term hardening target, not a hard v1 blocker.** Rationale: containers give strong-enough isolation for the Pilot's scale and known-participant risk profile, provided the scanning pipeline (Document 01 §2) and network-denial-by-default policy (Section 4 below) are both in place as compensating controls. This should be revisited before any anonymous, self-serve provider onboarding (Document 00's graduation trigger) since untrusted-workload risk increases materially at that point.

## 4. Network Policy
- Default: **no network access** inside the sandbox, unless the manifest's declared environment explicitly requires it (e.g., a job that needs to fetch a public dataset).
- Any granted network access is an explicit allowlist, not a blanket "internet access" toggle — minimizes both malicious exfiltration risk and accidental resource abuse.

## 5. Filesystem Layout Per Job
```
/environment (read-only, hash-verified, shared cache — Document 10)
/scratch (read-write, job-exclusive, wiped on completion)
```
No other mounts. No access to the host filesystem, other jobs' scratch spaces, or the Worker Agent's own credential storage.

## 6. Lifecycle (ties to Document 08 §3.3)
Create → mount environment (read-only) + fresh scratch → execute → stage-1 validation → upload result → wipe scratch → destroy sandbox → idle.

## 7. Compromised-Sandbox Containment
Even if a job's code is malicious and evades scanning (Document 01 §2's honest limitation), the isolation boundaries above are designed to contain the blast radius to: that job's own scratch space (wiped afterward) and whatever narrow network allowlist it was granted (default: none). It should not be able to affect the host, other jobs, or the Worker Agent itself.

## Senior Review
**Security Notes:** This document's container-vs-microVM decision is the one most likely to need revisiting as the platform scales past known/trusted Pilot participants — flagged explicitly rather than silently assumed permanent.
**Risks:** Container escapes (kernel-level vulnerabilities) are a known, real attack class — compensating controls (scanning, network denial-by-default, resource limits) are necessary specifically because container isolation alone is not absolute.
**Suggested Future Improvement:** Benchmark microVM overhead against container overhead on realistic Pilot workloads before committing to a v2 migration — the decision should be data-driven, not purely theoretical.
