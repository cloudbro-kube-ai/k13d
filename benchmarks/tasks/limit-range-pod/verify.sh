#!/bin/bash
set -e

echo "Verifying limit-range-pod task..."

NAMESPACE="limitrange-pod-test"

# Check LimitRange exists
if ! kubectl get limitrange pod-limits -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: LimitRange 'pod-limits' not found"
    exit 1
fi

# Check LimitRange has Pod type
LIMIT_TYPE=$(kubectl get limitrange pod-limits -n "$NAMESPACE" -o jsonpath='{.spec.limits[*].type}' 2>/dev/null || echo "")
if [[ "$LIMIT_TYPE" != *"Pod"* ]]; then
    echo "FAILED: LimitRange should have type 'Pod', got '$LIMIT_TYPE'"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod test-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'test-pod' not found"
    exit 1
fi

# Check Pod resources are within limits
CPU_REQUEST=$(kubectl get pod test-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "")
if [[ -z "$CPU_REQUEST" ]]; then
    echo "FAILED: Pod should have CPU request set"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/test-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

echo "SUCCESS: LimitRange for Pod correctly configured"
exit 0
