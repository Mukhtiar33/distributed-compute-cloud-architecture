# Document 15 — Monitoring & Observability

## Objective & Scope
Specify the minimum observability required to operate and debug the Pilot safely, and what's deferred to scale.

## 1. Core Classification
| Component | Classification |
|---|---|
| Structured logging (every component) | CORE |
| Job-level tracing (submission → completion, every stage) | CORE |
| Worker health signals | CORE |
| Environment cache hit-rate tracking | CORE (validates Document 10's core performance claim) |
| Full distributed tracing infrastructure | SCALE PARAMETER |
| Dashboards, alerting pipelines | SCALE PARAMETER |
| SLO/SLA tracking | DEFERRED |

## 2. Why Observability Is Core Even at Small Scale
Without structured, job-level tracing, debugging the security/isolation model itself (Document 11, 12) is impossible — if a sandbox behaves unexpectedly, there must be a clear trail of what happened at each stage (scan result, scheduling decision, environment resolution, execution outcome, validation results) to diagnose whether it's a bug or an actual security event.

## 3. What Must Be Logged Per Job (minimum v1 schema)
- Job ID, consumer/provider/worker identifiers (Document 13's metering data overlaps here — should share an identifier scheme, not duplicate independently).
- Manifest contents (job type, environment, entrypoint).
- Static scan result, dynamic detonation result.
- Scheduling decision and ranking factors that led to it (feeds Document 07 §4's future ML training data).
- Environment resolution: cache hit or miss, layer hash used.
- Execution outcome: success/failure, duration, resource consumption.
- Three-stage output validation results (Document 01 §2 revision).
- For parallel jobs: per-batch status history, including any reassignments.

## 4. Worker Health Signals
- Periodic heartbeat (feeds Document 09 §5's reassignment triggers).
- Resource availability snapshot (feeds Document 08 §3.4's throttling behavior and Document 07's scheduling inputs).
- Cached environment inventory (feeds scheduling ranking, Document 07 §3).

## 5. v1 Operational Model
Given the Pilot's small scale, v1 monitoring can reasonably be: structured logs written to a central store, queryable manually or via simple scripts — not a full dashboard/alerting stack. The schema (Section 3) must be correct from day one even if the tooling to visualize it is minimal; the data is more valuable long-term than the dashboard.

## 6. Privacy Consideration (cross-reference: Document 12 §6, Document 13)
Logs should capture *metadata about execution* (what happened, when, with what outcome), not job payload content — avoids monitoring infrastructure becoming an unintended second copy of sensitive consumer data.

## Senior Review
**Risks:** The temptation in a small Pilot is to skip structured logging ("I'll just watch the terminal") — this would undermine the ability to debug isolation/security issues rigorously and should be resisted even at N=5 workers.
**Suggested Future Improvement:** Once Pilot data accumulates, evaluate whether the job-level trace schema (Section 3) is sufficient input for Document 07's future ML scheduler, or whether additional fields need to be added before v2 — cheaper to extend the schema now than to discover a gap after months of incomplete data collection.
