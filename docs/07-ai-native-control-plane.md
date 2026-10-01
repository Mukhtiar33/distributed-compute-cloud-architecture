# Document 07 — AI-Native Control Plane

## Objective & Scope
Define what "AI-native" means concretely, and specify the v1 heuristic implementation that sits behind the same interface a future ML-driven system will use.

## 1. Core Classification
| Component | Classification |
|---|---|
| Decision interface (job → worker matching contract) | CORE |
| Heuristic decision logic (v1) | SCALE PARAMETER — replaceable implementation |
| ML-trained decision logic (v2+) | FUTURE, same interface |
| Data collection hooks feeding future ML training | CORE — must be built now even though the model doesn't exist yet |

## 2. Why "AI-Native" Doesn't Mean "ML From Day One"
An AI-native Control Plane is defined by its **decision interface** — a function that takes (job characteristics, worker pool state, historical performance) and returns a scheduling decision — not by what algorithm currently implements that function. Building a heuristic implementation behind the correct interface now means swapping in a trained model later is a deployment change, not an architectural one.

## 3. v1 Heuristic Scheduler
Inputs available in v1:
- Declared environment requirement (manifest).
- Worker capability advertisement (what environments a worker has cached, hardware profile).
- Worker current load/availability.
- Historical success/failure rate per worker (fed by the metering ledger, Document 03).

v1 decision logic: filter workers to those compatible with the job's environment and currently available → rank by (a) already having the required environment cached (avoids cold provisioning) → (b) historical reliability → (c) current load. Select highest-ranked.

## 4. Data Collection for Future ML (must be built in v1)
Every scheduling decision and its eventual outcome (success, failure, duration, resource consumption) must be logged in a structured, ML-trainable format from v1 onward — even though no model consumes it yet. Retrofitting this logging after the fact would lose the Pilot's entire decision history as training data.

Fields to capture per decision: job characteristics, candidate worker pool state at decision time, decision made, outcome, actual resource consumption vs. any estimate.

## 5. Prediction vs. Reaction (future direction, named not designed)
The long-term vision (Document 00) is a Control Plane that predicts resource availability rather than reacting to current state — e.g., anticipating that a given worker historically goes offline every evening. This requires the historical dataset from Section 4 to exist first; it is explicitly not a v1 capability.

## Senior Review
**Risks:** If Section 4's logging is skipped "since we don't have ML yet," the Pilot produces zero usable training data, and v2's ML scheduler starts from nothing despite months of Pilot operation.
**Suggested Future Improvement:** Once enough Pilot history accumulates, evaluate whether even a simple supervised model (predicting job success probability per worker) outperforms the hand-tuned heuristic ranking before investing in anything more sophisticated.
