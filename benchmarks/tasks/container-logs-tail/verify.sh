#!/bin/bash
set -e

echo "Verifying container-logs-tail task..."

NAMESPACE="logs-tail-test"

# Check Pod exists and is running
STATUS=$(kubectl get pod log-generator -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

# Verify logs can be retrieved with --tail
echo "Testing --tail option..."
LOGS_TAIL=$(kubectl logs log-generator -n "$NAMESPACE" --tail=10 2>/dev/null || echo "")
if [[ -z "$LOGS_TAIL" ]]; then
    echo "FAILED: Could not retrieve logs with --tail"
    exit 1
fi

# Count lines (should be 10 or less)
LINE_COUNT=$(echo "$LOGS_TAIL" | wc -l | tr -d ' ')
if [ "$LINE_COUNT" -gt 10 ]; then
    echo "FAILED: --tail=10 should return at most 10 lines, got $LINE_COUNT"
    exit 1
fi

# Verify logs can be retrieved with --since
echo "Testing --since option..."
LOGS_SINCE=$(kubectl logs log-generator -n "$NAMESPACE" --since=30s 2>/dev/null || echo "")
if [[ -z "$LOGS_SINCE" ]]; then
    echo "WARNING: --since=30s returned no logs (may be timing issue)"
fi

# Verify logs can be retrieved with --timestamps
echo "Testing --timestamps option..."
LOGS_TS=$(kubectl logs log-generator -n "$NAMESPACE" --tail=5 --timestamps 2>/dev/null || echo "")
if [[ "$LOGS_TS" != *"Z"* ]]; then
    echo "WARNING: --timestamps may not be showing timestamps"
fi

echo "SUCCESS: Container logs commands verified"
exit 0
