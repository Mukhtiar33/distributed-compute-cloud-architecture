# Document 03 — Business Model

## Objective & Scope
Define how the platform sustains itself economically, at a level of detail appropriate to v1 (where monetization is largely deferred per Document 00 §3.6) while keeping the long-term model coherent.

## 1. Core Classification
| Item | Classification |
|---|---|
| Job metering (resource consumption tracked per consumer/provider pair) | CORE (v1) |
| Pricing model, billing, payouts | DEFERRED |
| Marketplace matching / bidding | DEFERRED |
| Provider incentive structure | DEFERRED, but must not be foreclosed by v1 metering design |

## 2. Long-Term Revenue Model (directional, not committed)
- **Consumer side:** pay for compute consumed (credits or usage-based billing), analogous to existing cloud compute pricing but sourced from idle capacity rather than owned datacenters.
- **Provider side:** earn credits/revenue share for contributed compute, proportional to verified, successfully-completed work (tied to the metering seam established in v1).
- **Platform margin:** spread between what consumers pay and what providers earn, plus potential premium tiers (priority scheduling, guaranteed environments, SLA-backed execution) once the platform matures.

## 3. Why Metering Is Core Even Though Pricing Isn't
Retrofitting usage tracking into an execution pipeline that wasn't built to record it is expensive and, worse, produces incomplete historical data for anything that ran before the retrofit. Document 01 already requires the Control Plane to attribute every job to a consumer/provider pair and validate outputs in three stages — metering rides on the same data path at near-zero marginal cost if captured now.

## 4. v1 Economic Model
- No real payment processing, no live pricing.
- Every job still records: consumer identity, provider/worker identity, resource consumption (CPU/GPU time, memory, duration), and success/failure status — the exact ledger a future economy will need, populated from day one.
- Providers in the Pilot participate for non-monetary reasons (helping test, personal relationship) — this is explicitly a Pilot-only condition and must not be architecturally assumed anywhere outside this document.

## 5. Open Questions (for future resolution)
- Credit-based internal currency vs. direct fiat billing.
- How reliability/reputation (Document 06) factors into provider compensation.
- Whether pricing is fixed-rate, dynamic/market-based, or tiered by SLA.

## Senior Review
**Risks:** If metering is skipped in v1 "since there's no money yet," the Pilot's job history becomes economically worthless later — this is the one item in this document that's actually CORE despite the rest being deferred.
**Technical Debt:** None if metering is captured now; significant if deferred alongside pricing.
