#!/bin/bash
set -e

echo "Verifying ambassador-pattern task..."

NAMESPACE="ambassador-test"

# Check Pod exists
if ! kubectl get pod ambassador-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'ambassador-pod' not found"
    exit 1
fi

# Check both containers exist
CONTAINERS=$(kubectl get pod ambassador-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")
if [[ "$CONTAINERS" != *"app"* ]]; then
    echo "FAILED: Container 'app' not found"
    exit 1
fi
if [[ "$CONTAINERS" != *"redis-ambassador"* ]]; then
    echo "FAILED: Container 'redis-ambassador' not found"
    exit 1
fi

# Check redis-ambassador uses redis image
REDIS_IMAGE=$(kubectl get pod ambassador-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[?(@.name=="redis-ambassador")].image}' 2>/dev/null || echo "")
if [[ "$REDIS_IMAGE" != *"redis"* ]]; then
    echo "FAILED: Ambassador container should use redis image, got '$REDIS_IMAGE'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/ambassador-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Check app can connect to redis via localhost
CONNECTION_TEST=$(kubectl exec ambassador-pod -n "$NAMESPACE" -c app -- nc -z localhost 6379 2>&1 && echo "connected" || echo "failed")
if [[ "$CONNECTION_TEST" != *"connected"* ]]; then
    echo "WARNING: App may not be able to connect to redis ambassador"
fi

echo "SUCCESS: Ambassador pattern correctly configured"
exit 0
