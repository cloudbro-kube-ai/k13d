#!/bin/bash
set -e

echo "Verifying init-container-dependency task..."

NAMESPACE="init-dep-test"

# Check Redis deployment exists
if ! kubectl get deployment redis -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Deployment 'redis' not found"
    exit 1
fi

# Check Redis service exists
if ! kubectl get service redis -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Service 'redis' not found"
    exit 1
fi

# Check service port
SERVICE_PORT=$(kubectl get service redis -n "$NAMESPACE" -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo "")
if [ "$SERVICE_PORT" != "6379" ]; then
    echo "FAILED: Redis service should expose port 6379, got '$SERVICE_PORT'"
    exit 1
fi

# Check app Pod exists
if ! kubectl get pod app-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'app-pod' not found"
    exit 1
fi

# Check init container exists with correct name
INIT_CONTAINERS=$(kubectl get pod app-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null || echo "")
if [[ "$INIT_CONTAINERS" != *"check-redis"* ]]; then
    echo "FAILED: Init container 'check-redis' not found"
    exit 1
fi

# Wait for pod to be ready (all init containers completed)
if ! kubectl wait --for=condition=Ready pod/app-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Check pod is running
STATUS=$(kubectl get pod app-pod -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

echo "SUCCESS: Init container dependency check pattern correctly configured"
exit 0
