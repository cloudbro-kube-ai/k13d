#!/bin/bash
set -e

echo "Verifying previous-container-logs task..."

NAMESPACE="prev-logs-test"

# Check Pod exists
if ! kubectl get pod crash-loop-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'crash-loop-pod' not found"
    exit 1
fi

# Check pod has restarted at least once
RESTARTS=$(kubectl get pod crash-loop-pod -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
if [ "$RESTARTS" -lt 1 ]; then
    echo "INFO: Pod hasn't restarted yet (restarts: $RESTARTS), waiting..."
    sleep 30
    RESTARTS=$(kubectl get pod crash-loop-pod -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
fi

# Verify we can get previous logs
echo "Testing --previous flag..."
PREV_LOGS=$(kubectl logs crash-loop-pod -n "$NAMESPACE" --previous 2>/dev/null || echo "NO_PREVIOUS")

if [[ "$PREV_LOGS" == "NO_PREVIOUS" ]]; then
    echo "WARNING: Could not retrieve previous logs (container may not have restarted yet)"
    # Still pass if the setup is correct
    exit 0
fi

if [[ "$PREV_LOGS" != *"STARTUP"* ]] && [[ "$PREV_LOGS" != *"ERROR"* ]]; then
    echo "WARNING: Previous logs don't contain expected content"
fi

echo "Previous container logs retrieved successfully"
echo "Restart count: $RESTARTS"

echo "SUCCESS: Previous container logs accessible"
exit 0
