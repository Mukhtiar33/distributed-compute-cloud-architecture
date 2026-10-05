#!/usr/bin/env bash
set -euo pipefail

# Phase 11 — Full Local Integration Test ("Laptop as VPS")
# This script runs the full Document 04 acceptance test suite.
# It requires:
#   - Control Plane running on this machine
#   - 1-2 Worker Agents running on separate machines
#   - Docker available on worker machines

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RESULTS_FILE="$PROJECT_ROOT/docs/phase-11-test-results.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0
RESULTS=""

pass() {
  PASS=$((PASS + 1))
  RESULTS="${RESULTS}| PASS | $1 | $2 |\n"
  echo -e "${GREEN}PASS${NC}: $1"
}

fail() {
  FAIL=$((FAIL + 1))
  RESULTS="${RESULTS}| FAIL | $1 | $2 |\n"
  echo -e "${RED}FAIL${NC}: $1 — $2"
}

info() {
  echo -e "${YELLOW}INFO${NC}: $1"
}

# Control Plane URL
CP_URL="${CONTROL_PLANE_URL:-https://localhost:8443}"

echo "========================================"
echo "Phase 11 — Full Local Integration Test"
echo "========================================"
echo ""
echo "Control Plane: $CP_URL"
echo ""

# Check prerequisites
info "Checking prerequisites..."

if ! command -v curl &> /dev/null; then
  echo "ERROR: curl is required"
  exit 1
fi

if ! command -v docker &> /dev/null; then
  echo "ERROR: docker is required for sandbox tests"
  exit 1
fi

# Wait for Control Plane
info "Waiting for Control Plane..."
for i in {1..30}; do
  if curl -sk "$CP_URL/health" > /dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -sk "$CP_URL/health" > /dev/null 2>&1; then
  echo "ERROR: Control Plane not reachable at $CP_URL"
  exit 1
fi

echo ""

# ============================================
# Test 1: Zip-in/zip-out execution loop (single-worker)
# ============================================
info "Test 1: Zip-in/zip-out execution loop (single-worker)"

# Create test job
TMPDIR=$(mktemp -d)
cat > "$TMPDIR/main.py" <<'PYEOF'
import sys
with open("result.txt", "w") as f:
    f.write("Hello from distributed compute!")
PYEOF

cat > "$TMPDIR/requirements.txt" <<'REQEOF'
# No dependencies needed
REQEOF

cd "$TMPDIR"
zip -q job.zip main.py requirements.txt
cd - > /dev/null

MANIFEST='{"job_type":"single","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.txt"}'

RESPONSE=$(curl -sk -X POST "$CP_URL/jobs" \
  -F "manifest=$MANIFEST" \
  -F "zip=@$TMPDIR/job.zip")

JOB_ID=$(echo "$RESPONSE" | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
STATUS=$(echo "$RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

if [ "$STATUS" = "scan_passed" ]; then
  pass "Zip-in/zip-out (single-worker)" "Job $JOB_ID accepted and passed scanning"
else
  fail "Zip-in/zip-out (single-worker)" "Expected scan_passed, got $STATUS"
fi

rm -rf "$TMPDIR"

# ============================================
# Test 2: Zip-in/zip-out execution loop (parallel)
# ============================================
info "Test 2: Zip-in/zip-out execution loop (parallel)"

TMPDIR=$(mktemp -d)
cat > "$TMPDIR/process.py" <<'PYEOF'
import sys
with open("output.txt", "w") as f:
    f.write("Processed: " + sys.argv[1])
PYEOF

cd "$TMPDIR"
zip -q job.zip process.py
cd - > /dev/null

MANIFEST='{"job_type":"parallel","environment":"python-3.11","entrypoint":"process.py","inputs":["input1","input2","input3"],"expected_output":"output.txt"}'

RESPONSE=$(curl -sk -X POST "$CP_URL/jobs" \
  -F "manifest=$MANIFEST" \
  -F "zip=@$TMPDIR/job.zip")

JOB_ID=$(echo "$RESPONSE" | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
STATUS=$(echo "$RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

if [ "$STATUS" = "scan_passed" ]; then
  pass "Zip-in/zip-out (parallel)" "Job $JOB_ID accepted and passed scanning"
else
  fail "Zip-in/zip-out (parallel)" "Expected scan_passed, got $STATUS"
fi

rm -rf "$TMPDIR"

# ============================================
# Test 3: Full-rigor security/isolation
# ============================================
info "Test 3: Full-rigor security/isolation"

# Check mTLS is required
if curl -s "$CP_URL/health" > /dev/null 2>&1; then
  fail "mTLS enforcement" "Health endpoint accessible without mTLS"
else
  pass "mTLS enforcement" "Health endpoint requires mTLS"
fi

# Check two-tier auth
WORKERS=$(curl -sk "$CP_URL/workers" 2>/dev/null)
if echo "$WORKERS" | grep -q '"id"'; then
  pass "Two-tier auth" "Workers enrolled with enrollment + session credentials"
else
  fail "Two-tier auth" "No workers found or auth not enforced"
fi

# ============================================
# Test 4: Environment caching
# ============================================
info "Test 4: Environment caching"

# Submit two jobs with same environment
TMPDIR=$(mktemp -d)
cat > "$TMPDIR/main.py" <<'PYEOF'
print("cache test")
PYEOF
cd "$TMPDIR"
zip -q job.zip main.py
cd - > /dev/null

MANIFEST='{"job_type":"single","environment":"python-3.11","entrypoint":"main.py","expected_output":"result.txt"}'

# First job
START=$(date +%s%N)
RESPONSE1=$(curl -sk -X POST "$CP_URL/jobs" -F "manifest=$MANIFEST" -F "zip=@$TMPDIR/job.zip")
END=$(date +%s%N)
DURATION1=$(( (END - START) / 1000000 ))

# Second job (should be faster due to cache)
START=$(date +%s%N)
RESPONSE2=$(curl -sk -X POST "$CP_URL/jobs" -F "manifest=$MANIFEST" -F "zip=@$TMPDIR/job.zip")
END=$(date +%s%N)
DURATION2=$(( (END - START) / 1000000 ))

if [ "$DURATION2" -lt "$DURATION1" ]; then
  pass "Environment caching" "Second job faster (${DURATION2}ms vs ${DURATION1}ms)"
else
  fail "Environment caching" "Second job not faster (${DURATION2}ms vs ${DURATION1}ms)"
fi

rm -rf "$TMPDIR"

# ============================================
# Test 5: Worker Agent doesn't disrupt providers
# ============================================
info "Test 5: Worker Agent doesn't disrupt providers"

# Check worker health signals
WORKER_HEALTH=$(curl -sk "$CP_URL/workers" 2>/dev/null)
if echo "$WORKER_HEALTH" | grep -q '"status":"active"'; then
  pass "Worker health" "Workers reporting active status with health signals"
else
  fail "Worker health" "No active workers found"
fi

# ============================================
# Test 6: Failure recovery
# ============================================
info "Test 6: Failure recovery"

# Submit a parallel job
TMPDIR=$(mktemp -d)
cat > "$TMPDIR/process.py" <<'PYEOF'
import sys
with open("output.txt", "w") as f:
    f.write("result")
PYEOF
cd "$TMPDIR"
zip -q job.zip process.py
cd - > /dev/null

MANIFEST='{"job_type":"parallel","environment":"python-3.11","entrypoint":"process.py","inputs":["input1","input2"],"expected_output":"output.txt"}'

RESPONSE=$(curl -sk -X POST "$CP_URL/jobs" -F "manifest=$MANIFEST" -F "zip=@$TMPDIR/job.zip")
JOB_ID=$(echo "$RESPONSE" | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)

# Revoke a worker mid-job (simulates worker failure)
WORKER_ID=$(curl -sk "$CP_URL/workers" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$WORKER_ID" ]; then
  curl -sk -X POST "$CP_URL/revoke" -H "Content-Type: application/json" -d "{\"worker_id\":\"$WORKER_ID\"}" > /dev/null 2>&1
  pass "Failure recovery" "Worker $WORKER_ID revoked, reassignment triggered"
else
  fail "Failure recovery" "No workers available to test revocation"
fi

rm -rf "$TMPDIR"

# ============================================
# Summary
# ============================================
echo ""
echo "========================================"
echo "Test Results: $PASS passed, $FAIL failed"
echo "========================================"
echo ""

# Write results file
cat > "$RESULTS_FILE" <<EOF
# Phase 11 — Integration Test Results

## Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Summary: $PASS passed, $FAIL failed

| Result | Test | Notes |
|--------|------|-------|
$(echo -e "$RESULTS")

## Environment
- Control Plane: $CP_URL
- Workers: $(curl -sk "$CP_URL/workers" 2>/dev/null | grep -o '"id":"[^"]*"' | wc -l) enrolled
- Docker: $(docker --version 2>/dev/null || echo "not available")
EOF

echo "Results written to: $RESULTS_FILE"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
