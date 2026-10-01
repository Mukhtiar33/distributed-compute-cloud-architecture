# Document 14 — Credit Economy

## Objective & Scope
Detail the credit/incentive system's eventual design, building on the metering seam established as CORE in Document 03.

## 1. Core Classification
| Component | Classification |
|---|---|
| Metering (resource consumption tracked per job) | CORE (already specified — Document 03 §3, Document 13 §4) |
| Credit issuance, balances, spending | DEFERRED |
| Pricing (rates per resource-unit) | DEFERRED |
| Fraud/abuse prevention for the economy | DEFERRED, but should reuse the trust/security primitives from Document 12 rather than invent new ones |

## 2. Why This Document Exists Despite Being Mostly Deferred
Per Document 00's classification framework, "deferred" doesn't mean "undesigned forever" — it means the detailed mechanics aren't needed for the Pilot, but the *shape* of the system should be sketched now so v1's metering design doesn't accidentally foreclose it.

## 3. Directional Design (not committed, for future refinement)
- **Provider earns credits** proportional to successfully validated work (ties to the three-stage output validation in Document 01 §2 — only validated, non-fraudulent output should earn credit).
- **Consumer spends credits** proportional to resource consumption recorded by the metering ledger.
- **Reliability weighting:** a provider's historical success rate (already tracked for scheduling purposes, Document 07 §3) could plausibly factor into earn rate — rewarding consistently reliable providers — but this is a v2+ economic design decision, not a v1 architectural requirement.

## 4. Fraud/Abuse Considerations (named, not designed)
Once real value is attached to compute contribution, new incentives appear that didn't exist in the Pilot: providers might try to fake successful job completion, or consumers might try to under-report consumption. This document flags that the existing three-stage validation and metering-at-the-source design (Document 01, Document 03) are the right foundation to build economic fraud prevention on top of — but the specific anti-fraud mechanisms are out of scope until the economy itself is designed.

## 5. What Must NOT Be Retrofitted Later
- Metering must already be accurate and tamper-resistant by the time a real economy launches — this is why it's CORE now (Document 03 §3) rather than being added alongside pricing.
- Provider/consumer identity must already be stable and revocable (Document 01 §3) before credits can be meaningfully tied to identity.

## Senior Review
**Risks:** The single biggest risk to this future subsystem is *not* in this document — it's in Document 01/03's metering design being incomplete or inaccurate, which would undermine any economy built on top of it later. This document's real value is confirming those foundations are sufficient, which they are, based on the current design.
**Technical Debt:** Entire pricing/fraud-prevention layer is intentionally deferred — acceptable given v1's manually-onboarded, non-monetary Pilot participants (Document 03 §4).
