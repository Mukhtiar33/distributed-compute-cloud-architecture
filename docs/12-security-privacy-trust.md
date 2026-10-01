# Document 12 — Security, Privacy & Trust Model

## Objective & Scope
Consolidate the platform's Zero Trust posture across every subsystem into a single reference, cross-linking to the detailed mechanisms already specified elsewhere.

## 1. Governing Principle
No party trusts any other party by default — consumer, provider, and the platform's own internal components all operate under Zero Trust. This is Document 00 §1's rule restated as a cross-cutting policy.

## 2. Trust Matrix

| Actor | Trusts | Does Not Trust (by default) |
|---|---|---|
| Consumer | The platform to isolate their job and protect their data | Any provider, any other consumer |
| Provider | The platform to isolate jobs running on their machine and not damage it | Any consumer's submitted code, any other provider |
| Control Plane | Nothing by default | Consumer submissions (Document 01 §2), worker outputs (three-stage validation), workers' identity claims (verified via mTLS + session tokens, Document 01 §3) |

## 3. Authentication & Identity (cross-reference: Document 01 §3, Document 08 §3.1)
- Two-tier credential model: enrollment (once, keypair-bound) + session (short-lived, auto-refreshed).
- Mutual TLS for transport-level identity, independent of the token layer.
- Immediate revocation path for flagged workers.

## 4. Workload Vetting (cross-reference: Document 01 §2)
- Static scanning + sandboxed dynamic detonation, both mandatory, neither sufficient alone.
- Explicit acknowledgment: this catches known/behaviorally-anomalous threats, not all possible novel attacks — isolation (Document 11) is the backstop, not scanning.

## 5. Isolation (cross-reference: Document 01 §4, Document 11)
- Process, filesystem, network, memory, secrets — all per-job, all mandatory, unaffected by scale.

## 6. Data Privacy
- No consumer data persists on provider machines after job completion (scratch space wiped, Document 11 §6).
- No provider data is accessible to workloads beyond the scoped, read-only environment cache.
- Metering data (Document 03) records resource consumption, not job *content* — the ledger should be designed to avoid inadvertently becoming a second copy of sensitive consumer data.

## 7. Output Trust (cross-reference: Document 01 §2 revision)
Three-stage validation: worker-local sanity check → Control Plane authoritative validation → pre-delivery integrity check. No stage is trusted to catch what another stage is responsible for.

## 8. Threat Model Summary

| Threat | Primary Mitigation | Compensating Control |
|---|---|---|
| Malicious consumer job | Static + dynamic scanning | Sandbox isolation contains any evasion |
| Compromised/malicious worker | Three-stage output validation, per-job isolation preventing cross-job access | Reputation/reliability tracking feeds future scheduling decisions (Document 07) |
| Credential theft (worker) | Short-lived session tokens, mTLS | Immediate revocation, re-enrollment friction |
| Environment cache tampering | Hash verification on every mount, not just download | Read-only mount prevents in-place modification |
| Hybrid-user role abuse (Document 05 §3) | Separate consumer/provider identities, no self-preferential routing | — |

## 9. Failure Modes & Recovery
- A worker fails scanning/validation repeatedly → flagged, session revoked, requires re-enrollment (higher friction than routine operation).
- A consumer's job repeatedly trips dynamic detonation anomalies → job rejected, submission flagged for review (specific review process TBD as an open item — see Section 10).
- Control Plane itself compromised → out of scope for this document; addressed structurally by Document 06's single-point-of-coordination risk being an accepted, documented v1 limitation.

## 10. Open Questions
- What is the concrete review/appeals process when a legitimate job is incorrectly flagged by dynamic detonation (false positive handling)?
- What is the retention policy for scan/validation logs — how long are they kept, and are they themselves treated as sensitive data?

## Senior Review
**Risks:** This document is deliberately a consolidation, not new design — its main risk is drifting out of sync with Documents 01/08/10/11 as those evolve. It should be revisited whenever any of its cross-referenced sections change.
**Security Notes:** Section 8's threat/mitigation table is the most implementation-actionable part of this document — recommend using it directly as a checklist during Pilot security testing.
