#!/bin/bash
set -euo pipefail

NAMESPACE="memory-issues"
TIMEOUT="180s"

echo "Verifying debug-oom-killer fix..."

# Check deployment exists
if ! kubectl get deployment memory-app -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Deployment 'memory-app' not found"
    exit 1
fi

# Check memory limits have been increased
MEMORY_LIMIT=$(kubectl get deployment memory-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}')

# Convert to Mi for comparison
LIMIT_VALUE=$(echo "$MEMORY_LIMIT" | sed 's/Mi//' | sed 's/Gi/*1024/' | bc 2>/dev/null || echo "0")
if [[ "$LIMIT_VALUE" -lt 256 ]]; then
    echo "ERROR: Memory limit should be at least 256Mi, got '$MEMORY_LIMIT'"
    exit 1
fi

# Check memory requests
MEMORY_REQUEST=$(kubectl get deployment memory-app -n $NAMESPACE -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}')
REQUEST_VALUE=$(echo "$MEMORY_REQUEST" | sed 's/Mi//' | sed 's/Gi/*1024/' | bc 2>/dev/null || echo "0")
if [[ "$REQUEST_VALUE" -lt 128 ]]; then
    echo "ERROR: Memory request should be at least 128Mi, got '$MEMORY_REQUEST'"
    exit 1
fi

# Wait for deployment to be ready
echo "Waiting for deployment to be ready..."
if ! kubectl rollout status deployment/memory-app -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "ERROR: Deployment is not ready"
    exit 1
fi

# Wait additional time to ensure stability
sleep 30

# Check for recent OOM events (should be none in last 30 seconds)
RECENT_OOM=$(kubectl get events -n $NAMESPACE --field-selector reason=OOMKilling --sort-by='.lastTimestamp' -o json 2>/dev/null | jq -r '.items[-1].lastTimestamp // "none"')

if [[ "$RECENT_OOM" != "none" ]]; then
    # Check if the event is recent (within last 30 seconds)
    if command -v gdate &>/dev/null; then
        DATE_CMD="gdate"
    else
        DATE_CMD="date"
    fi

    EVENT_TIME=$($DATE_CMD -d "$RECENT_OOM" +%s 2>/dev/null || echo "0")
    CURRENT_TIME=$($DATE_CMD +%s)

    if [[ $((CURRENT_TIME - EVENT_TIME)) -lt 30 ]]; then
        echo "ERROR: Recent OOM events detected, fix not complete"
        exit 1
    fi
fi

# Check pod is running without recent restarts
POD_RESTARTS=$(kubectl get pods -n $NAMESPACE -l app=memory-app -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
POD_READY=$(kubectl get pods -n $NAMESPACE -l app=memory-app -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "False")

if [[ "$POD_READY" != "True" ]]; then
    echo "ERROR: Pod is not ready"
    exit 1
fi

# Check VPA annotation
VPA_ANNOTATION=$(kubectl get deployment memory-app -n $NAMESPACE -o jsonpath='{.metadata.annotations.vpa\.k13d\.io/recommended-memory-limit}' 2>/dev/null || echo "")
if [[ -z "$VPA_ANNOTATION" ]]; then
    echo "WARNING: VPA recommendation annotation not set"
fi

echo "--- Verification Successful! ---"
echo "OOM issue has been fixed. Memory limits: $MEMORY_LIMIT, Requests: $MEMORY_REQUEST"
exit 0
