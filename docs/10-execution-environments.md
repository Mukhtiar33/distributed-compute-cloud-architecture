# Document 10 — Execution Environments

## Objective & Scope
Specify the environment registry mechanism, the manifest schema that drives it, and the v1 catalog scope.

## 1. Core Classification
| Component | Classification |
|---|---|
| Immutable, content-addressed layer mechanism | CORE |
| Hash verification on every mount (not just download) | CORE |
| Manifest schema | CORE (minimal v1 fields, extensible) |
| Catalog breadth (number of supported frameworks) | SCALE PARAMETER |

## 2. Environment Layer Model
- Environments are packaged as immutable, versioned, content-addressed layers (conceptually similar to container image layers).
- Each layer is identified by a hash of its contents; the Worker Agent verifies this hash on every mount, not only at initial download, to defend against local tampering on a compromised worker's disk.
- Layers are read-only when mounted into a sandbox — no job can write to or modify the shared cache (Document 01 §6, §4).

## 3. Manifest Schema (v1 minimum)
```
job_type: single | parallel
environment: <registry identifier>
entrypoint: <what to execute>
inputs: [<list, only for job_type: parallel>]
expected_output: <minimal shape descriptor>
```
This schema is intentionally minimal for v1 and must be designed for additive extension (new optional fields) rather than breaking changes, since v1 Pilot jobs and their manifests may need to remain valid as the schema grows.

## 4. Environment Resolution Flow
1. Worker Agent receives job with a declared `environment` identifier.
2. Checks local cache for a hash-verified, matching layer.
3. Cache hit: mount directly, no network activity required.
4. Cache miss: pull from Environment Registry, verify hash, cache, then mount.

## 5. v1 Catalog Scope
Deliberately narrow — limited to the specific workload types the Pilot is designed to prove (per Document 04's goals). Exact framework list is an implementation decision made alongside the Pilot's first real test jobs, not fixed permanently by this document.

## 6. Cache Invalidation (must be specified, previously flagged as open)
- When an environment layer is updated (new version published to the registry), existing cached copies on workers are **not** silently overwritten — the new version gets a new content hash and is treated as a distinct layer.
- Jobs specify environment by identifier, which the registry resolves to a specific version (default: latest, or pinned if the manifest specifies a version) — this keeps in-flight jobs deterministic even if the registry is updated mid-flight.
- Stale/unused cached layers on a worker may be garbage-collected locally (e.g., least-recently-used eviction) when local storage is constrained — this is a Worker Agent implementation detail, not a correctness concern, since any evicted layer can always be re-pulled and re-verified.

## Senior Review
**Risks:** Without the versioning/pinning behavior in Section 6, environment updates could cause non-deterministic behavior for in-flight jobs — this was an open question in Document 01 and is now resolved here.
**Performance Notes:** Cache hit rate is the key performance metric for this subsystem — should be tracked from v1 onward (ties into Document 15's monitoring requirements) to validate the caching design is actually delivering the speedup it's meant to.
