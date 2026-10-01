# Document 09 — Scheduler & Intelligent Orchestration

## Objective & Scope
Formalize the scheduling/orchestration logic referenced in Documents 01 and 07, covering job intake through completion for both job types.

## 1. Core Classification
| Component | Classification |
|---|---|
| Job intake, validation, environment resolution | CORE |
| Single-worker job assignment | CORE |
| Independent-parallel batch grouping & assignment | CORE |
| Batch-level status tracking & reassignment | CORE |
| Decision ranking logic | SCALE PARAMETER (heuristic v1, ML later — Document 07) |

## 2. Job Intake Pipeline
1. Manifest parsed and validated (required fields present, environment identifier known to the registry).
2. Static + dynamic scanning (Document 01 §2).
3. Job classified as single-worker or independent-parallel based on manifest's `job_type` and presence of an input list.
4. Enters the scheduling queue.

## 3. Single-Worker Assignment
- Scheduler applies the ranking logic from Document 07 §3 to select one worker.
- Job is bound to that worker until completion or failure-triggered reassignment.
- Framework-native checkpoint capture (Document 01 §5.1) is enabled by default for long-running job types (e.g., training) to bound reassignment cost.

## 4. Independent-Parallel Batch Grouping
- Control Plane groups the manifest's declared inputs into batches — grouping strategy in v1: simple round-robin/capacity-based grouping across currently available compatible workers, not consumer-influenced.
- Each batch is scheduled independently via the same single-worker assignment logic (Section 3), tracked under a shared parent job ID.
- Batch status model: `pending → running → (succeeded | failed) → [reassigned if failed]`.

## 5. Failure & Reassignment
- Health-check timeout on a worker triggers reassignment of its in-flight job/batch to another compatible worker.
- Reassignment is idempotent: a batch that appears to succeed after being marked failed-and-reassigned (e.g., a late/duplicate result from the original worker) must not be double-counted or cause data corruption — the parent job tracks a single authoritative result per batch, keyed by batch ID, with late/duplicate submissions discarded once a batch is already marked succeeded.
- Single-worker jobs use the same mechanism at the whole-job level, mitigated by checkpoint resume (Document 08 §3.3) where available.

## 6. Completion & Assembly
- Single-worker: worker's validated output is the final result.
- Independent-parallel: once all batches reach `succeeded`, Control Plane performs stage-3 validation (Document 01 §2) across the full set before assembling the final result zip.

## 7. What This Scheduler Explicitly Does Not Do (v1)
- No distributed gradient synchronization across workers for a single training job (Document 01 §5.2).
- No predictive pre-warming or multi-region-aware placement (Document 00 §3.8).
- No consumer-influenced grouping decisions of any kind.

## Senior Review
**Risks:** The idempotency requirement in Section 5 (late/duplicate batch results) is a classic distributed-systems edge case that's easy to design correctly on paper and get wrong in implementation — should have dedicated test cases (simulate a "zombie" worker reporting success after being marked failed).
**Technical Debt:** v1's round-robin/capacity batch grouping is intentionally simple; smarter grouping (e.g., by data locality or worker specialization) is a natural, low-risk future improvement once the heuristic-vs-ML question (Document 07) is revisited.
