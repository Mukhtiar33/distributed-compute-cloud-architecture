# Phase 3 — Static Scanning Pipeline: Completion Report

## Status: COMPLETE

## What Was Built

### Deliverables (mapped to Document 17, Phase 3)

| Deliverable | Status | Files |
|---|---|---|
| Dependency vulnerability scanner | Done | `control-plane/internal/scanner/scanner.go` |
| Static code pattern checks | Done | `control-plane/internal/scanner/scanner.go` |
| Job status extended | Done | `control-plane/internal/jobs/store.go` |
| Scanner integration into job intake | Done | `control-plane/internal/api/jobs.go` |
| Test fixtures | Done | `control-plane/testdata/fixtures/` |

### Job Status Model (extended)

```
intake_validated → scanning → scan_passed | scan_rejected
```

### Vulnerability Database

| Package | Versions | Severity | CVE |
|---|---|---|---|
| requests | 2.25.0, 2.25.1, 2.24.0 | HIGH | CVE-2023-32681 |
| django | 3.2.0–3.2.5 | CRITICAL | CVE-2021-33203 |
| flask | 1.0.0–1.1.0 | MEDIUM | CVE-2023-30861 |
| lodash | 4.17.15–4.17.20 | HIGH | CVE-2021-23337 |
| express | 4.16.0–4.17.1 | MEDIUM | CVE-2022-24999 |

### Pattern Checks

- eval/exec with string literals
- os.system() calls
- subprocess execution
- base64 decoding (obfuscation)
- Dynamic os import
- ctypes.CDLL loading
- Java Runtime.exec
- Node.js child_process.exec

## Exit Criteria Verification

| Criterion | Result |
|---|---|
| A submission with a known-vulnerable dependency is flagged and blocked | PASS |
| A clean submission passes through to scan_passed | PASS |

## Test Results

### Unit Tests (49 tests, all passing)

```
control-plane/internal/scanner:
  TestScanner_CleanPayload — PASS
  TestScanner_VulnerableDependency — PASS
  TestScanner_DangerousPattern — PASS
  TestScanner_EntrypointExists — PASS
  TestScanner_EntrypointMissing — PASS
  TestScanner_InvalidZip — PASS
  TestScanner_EmptyZip — PASS
  TestScanner_MultipleVulnerabilities — PASS
  TestScanner_PackageJSONVulnerability — PASS
  TestScanner_CleanPackageJSON — PASS
  TestScanner_Base64Obfuscation — PASS
  TestScanner_GoModVulnerability — PASS
  TestVulnerabilityDB_CheckPythonRequirements — PASS
  TestVulnerabilityDB_CheckPackageJSON — PASS
  TestVulnerabilityDB_CleanDependencies — PASS

control-plane/internal/api: (15 tests — updated for scanner integration)
control-plane/internal/jobs: (3 tests — from Phase 2)
control-plane/internal/manifest: (11 tests — from Phase 2)
control-plane/internal/auth: (11 tests — from Phase 1)
control-plane/internal/config: (4 tests — from Phase 0)
```

### CI Results

```
Website (Next.js): PASS (lint + build)
Control Plane (Go): PASS (lint + build + test)
Worker Agent (Go): PASS (lint + build + test)
```

## Deviations from Documents 00–18

None. The implementation follows the static scanning requirements from Document 01 §2.

## Next Phase

Phase 4 — Sandbox Execution (Single-Worker Jobs, Full Isolation)
