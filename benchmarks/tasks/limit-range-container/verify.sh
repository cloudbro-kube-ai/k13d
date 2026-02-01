#!/bin/bash
set -e

echo "Verifying limit-range-container task..."

NAMESPACE="limitrange-container-test"

# Check LimitRange exists
if ! kubectl get limitrange container-limits -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: LimitRange 'container-limits' not found"
    exit 1
fi

# Check LimitRange has Container type
LIMIT_TYPE=$(kubectl get limitrange container-limits -n "$NAMESPACE" -o jsonpath='{.spec.limits[*].type}' 2>/dev/null || echo "")
if [[ "$LIMIT_TYPE" != *"Container"* ]]; then
    echo "FAILED: LimitRange should have type 'Container', got '$LIMIT_TYPE'"
    exit 1
fi

# Check default values exist in LimitRange
DEFAULT_CPU=$(kubectl get limitrange container-limits -n "$NAMESPACE" -o jsonpath='{.spec.limits[0].default.cpu}' 2>/dev/null || echo "")
if [[ -z "$DEFAULT_CPU" ]]; then
    echo "FAILED: LimitRange should have default CPU limit set"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod default-resources-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'default-resources-pod' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/default-resources-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Verify that default resources were applied to the pod
POD_CPU_LIMIT=$(kubectl get pod default-resources-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.limits.cpu}' 2>/dev/null || echo "")
if [[ -z "$POD_CPU_LIMIT" ]]; then
    echo "FAILED: Pod should have CPU limit applied by LimitRange defaults"
    exit 1
fi

echo "SUCCESS: LimitRange for Container correctly configured with defaults"
exit 0
