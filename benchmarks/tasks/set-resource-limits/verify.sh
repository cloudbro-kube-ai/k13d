#!/bin/bash
set -e

echo "Verifying set-resource-limits task..."

NAMESPACE="resource-lim-test"

# Check Pod exists
if ! kubectl get pod limited-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'limited-pod' not found"
    exit 1
fi

# Check CPU request
CPU_REQUEST=$(kubectl get pod limited-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "")
if [[ "$CPU_REQUEST" != "50m" ]]; then
    echo "FAILED: CPU request should be '50m', got '$CPU_REQUEST'"
    exit 1
fi

# Check Memory request
MEM_REQUEST=$(kubectl get pod limited-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.requests.memory}' 2>/dev/null || echo "")
if [[ "$MEM_REQUEST" != "64Mi" ]]; then
    echo "FAILED: Memory request should be '64Mi', got '$MEM_REQUEST'"
    exit 1
fi

# Check CPU limit
CPU_LIMIT=$(kubectl get pod limited-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.limits.cpu}' 2>/dev/null || echo "")
if [[ "$CPU_LIMIT" != "200m" ]]; then
    echo "FAILED: CPU limit should be '200m', got '$CPU_LIMIT'"
    exit 1
fi

# Check Memory limit
MEM_LIMIT=$(kubectl get pod limited-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].resources.limits.memory}' 2>/dev/null || echo "")
if [[ "$MEM_LIMIT" != "256Mi" ]]; then
    echo "FAILED: Memory limit should be '256Mi', got '$MEM_LIMIT'"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod limited-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$IMAGE" != *"httpd"* ]]; then
    echo "FAILED: Image should be httpd, got '$IMAGE'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/limited-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

echo "SUCCESS: Resource limits correctly configured"
exit 0
