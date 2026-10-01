# Document 16 — Scaling Philosophy

## Objective & Scope
Define how the platform grows from Pilot to Production without re-architecting the core (Document 00's central promise), and name the concrete graduation triggers for each deferred item across this document series.

## 1. Governing Rule (restated from Document 00)
Scaling changes throughput, geography, and population. It does not change trust boundaries, isolation guarantees, or the Control Plane's coordination contract.

## 2. Graduation Triggers by Subsystem

| Subsystem | v1 State | Graduation Trigger | v2+ Direction |
|---|---|---|---|
| Control Plane | Single logical instance (Document 06 §5) | Single-instance throughput saturates | Horizontal scaling — requires internal state to already be externalized (flagged in Doc 00 §5, must be verified as v1 is built) |
| Scheduler intelligence | Heuristic ranking (Document 07 §3) | Sufficient historical decision/outcome data accumulated (Document 07 §4) | ML-trained model behind the same decision interface |
| Provider onboarding | Manual, known individuals | Manual onboarding doesn't scale past N known providers | Self-serve registration — this is where identity/reputation systems become load-bearing (Document 05, Document 12 §8) and isolation assumptions (Document 11 §3's container-vs-microVM tradeoff) should be re-examined |
| Environment catalog | Narrow (Document 10 §5) | Consumer demand for unsupported frameworks | Broaden catalog — mechanism (Document 10 §2) is already general, only breadth grows |
| Sandboxing mechanism | Containers, baseline (Document 11 §3) | Anonymous/untrusted provider population increases risk profile | Migrate to microVM-level isolation |
| Credit economy | Metering only, no pricing (Document 14) | Platform ready to launch real economic incentives | Full pricing/billing/payout system, built on the existing metering foundation |
| Deployment footprint | Single region | Latency/demand from other regions | Multi-region — requires resolving Document 13 §5's cross-border data handling first |
| Distributed training | Not supported (single-worker only, Document 01 §5.2) | Sufficient single-worker/parallel-job validation, dedicated design effort available | Distributed multi-worker training with gradient sync — new communication pattern, needs its own security review against the "workers never talk directly" principle |

## 3. What Never Changes
- Zero Trust posture (Document 12) — same rigor at any scale.
- Per-job isolation (Document 11) — same guarantees at any scale.
- Two-tier authentication model (Document 01 §3) — same mechanism, more instances.
- Three-stage output validation (Document 01 §2 revision) — same pipeline, more throughput.
- "Infrastructure is invisible" consumer philosophy (Document 02 §2) — the zip interface may eventually be replaced by natural language, but the underlying manifest-driven contract does not change (Document 01 §1.3).

## 4. Scaling Anti-Patterns to Avoid
- Relaxing isolation "temporarily" to handle a throughput spike — never acceptable; scale the Control Plane/worker pool instead.
- Treating self-serve provider onboarding as a pure growth feature without revisiting the sandboxing mechanism (Section 2) — the risk profile genuinely changes when providers are no longer known individuals.
- Allowing the metering ledger (Document 03, 13) to be redesigned at economy-launch time instead of extended — this is exactly the retrofit cost Document 03 was written to avoid.

## Senior Review
**Risks:** This document's value depends entirely on the rest of the series holding its line under real implementation pressure — the graduation triggers above are only meaningful if v1 is actually built without shortcuts, per every prior document's Senior Review sections.
**Scalability Notes:** The Control Plane state-externalization question (Section 2, row 1) is the single highest-leverage thing to get right in v1's implementation, even though horizontal scaling itself is deferred — verify this specifically during Pilot implementation, not just at v2 planning time.
**Suggested Future Improvement:** Once the Pilot is running, revisit this document with real data (actual throughput ceilings, actual onboarding friction) rather than the current directional estimates.
