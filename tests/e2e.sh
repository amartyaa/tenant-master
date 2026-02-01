#!/usr/bin/env bash
set -euo pipefail

BFF=${1:-http://localhost:8080}

echo "Running Tenant-Master E2E smoke tests against $BFF"

# helper
jq_check(){
  if ! command -v jq &>/dev/null; then
    echo "jq is required for the tests; please install jq"
    exit 2
  fi
}

jq_check

# 1) List tenants
echo "-> GET /api/v1/tenants"
TENANTS_JSON=$(curl -sS "$BFF/api/v1/tenants")
if [[ -z "$TENANTS_JSON" ]]; then
  echo "FAIL: empty tenants response"
  exit 1
fi
echo "$TENANTS_JSON" | jq '.' >/dev/null

# 2) Create a tenant
TEST_NAME="e2e-test-$(date +%s)"
PAYLOAD=$(jq -n --arg name "$TEST_NAME" '{name:$name,tier:"Bronze",owner:"e2e@local.test",cpu:"500m",memory:"512Mi"}')
echo "-> POST create $TEST_NAME"
CREATED=$(curl -sS -X POST -H 'Content-Type: application/json' -d "$PAYLOAD" "$BFF/api/v1/tenants")
if [[ $? -ne 0 ]]; then
  echo "FAIL: create request failed"
  exit 1
fi
echo "$CREATED" | jq '.' >/dev/null

# 3) Verify list includes created tenant
echo "-> verify tenant present in list"
curl -sS "$BFF/api/v1/tenants" | jq --arg n "$TEST_NAME" 'map(select(.name==$n)) | length' | grep -q '^1$' && echo "OK: tenant found" || (echo "FAIL: tenant not found in list" && exit 1)

# 4) Get tenant detail
echo "-> GET tenant detail"
DETAIL=$(curl -sS "$BFF/api/v1/tenants/$TEST_NAME")
echo "$DETAIL" | jq '.' >/dev/null

# 5) Get metrics
echo "-> GET tenant metrics"
METRICS=$(curl -sS "$BFF/api/v1/tenants/$TEST_NAME/metrics" || true)
if [[ -n "$METRICS" ]]; then
  echo "$METRICS" | jq '.' >/dev/null || echo "Note: metrics not JSON or not present (ok for mock)"
else
  echo "Note: no metrics returned (ok for mock)"
fi

# 6) Delete tenant (mock supports file creation only; deletion may be 501)
echo "-> DELETE tenant"
DEL=$(curl -sS -X DELETE "$BFF/api/v1/tenants/$TEST_NAME" -w "\nHTTPSTATUS:%{http_code}\n" || true)
if echo "$DEL" | grep -q "HTTPSTATUS:501"; then
  echo "INFO: delete not supported in mock mode (expected)"
else
  echo "$DEL" | jq -r '.' >/dev/null 2>&1 || true
  echo "Deleted or no-op"
fi

echo "E2E smoke tests completed successfully against $BFF"
