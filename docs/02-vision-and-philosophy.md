# Document 02 — Project Vision & Philosophy

## Objective & Scope
Establish the enduring "why" behind the platform, independent of any specific v1 implementation choice. Every other document must remain consistent with this vision even as v1 scopes things down.

## 1. Mission
Transform idle personal computers into a secure, trustworthy distributed compute cloud — where consumers submit work without ever thinking about infrastructure, and providers contribute spare capacity without ever compromising their own machine's safety or usability.

## 2. Core Philosophy
- **Infrastructure is invisible to the consumer.** They describe or upload work; they never choose hardware, drivers, frameworks, or OS.
- **The provider's machine stays theirs.** The Worker Agent is a well-behaved guest — resource-throttled, sandboxed, reversible, never disruptive to normal use.
- **Trust is never assumed, on either side.** The platform is Zero Trust toward consumers, providers, and its own internal components. See Document 12 for the full trust model.
- **The Control Plane is the sole coordinator.** Workers never talk to each other; every interaction is mediated, observable, and auditable.
- **Nothing persists beyond its purpose.** Every workload's footprint is temporary by default; the only documented exception is the immutable, verified environment cache (Document 01 §6).

## 3. What Success Looks Like at Each Stage
- **Pilot (v1):** a handful of known providers, zip-in/zip-out consumer interface, full security/isolation/scheduling core intact, proving the execution model end-to-end.
- **v2 (Growth):** self-serve provider onboarding, broader environment catalog, ML-assisted scheduling behind the same interface, still single-region.
- **Production (v3+):** multi-region, horizontally scaled Control Plane, marketplace/credit economy live, natural-language job submission replacing the zip interface as the primary front door — same backend contract underneath (Document 01 §1.3).

## 4. What This Platform Is Not
- Not a general-purpose cloud VM rental service — workloads are sandboxed, ephemeral jobs, not persistent virtual machines the consumer administers.
- Not a platform that trusts "known" participants differently from anonymous ones — the Pilot's friends are treated with the same isolation rigor as a future anonymous stranger (Document 00 §1).

## Senior Review
**Risks:** The biggest risk to the vision is scope creep in the *opposite* direction — over-engineering v1 features that belong in v2/v3 (multi-region, full marketplace) at the expense of proving the core loop. Document 00/01's classification framework exists specifically to guard against this.
**Technical Debt:** None introduced by this document — it's purely a reference anchor for future decisions.
