#!/bin/bash
set -e

echo "Verifying init-container-wait task..."

NAMESPACE="init-wait-test"

# Check Service exists
if ! kubectl get service myservice -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Service 'myservice' not found"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod init-wait-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'init-wait-pod' not found"
    exit 1
fi

# Check init container exists with correct name
INIT_CONTAINERS=$(kubectl get pod init-wait-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null || echo "")
if [[ "$INIT_CONTAINERS" != *"wait-for-service"* ]]; then
    echo "FAILED: Init container 'wait-for-service' not found"
    exit 1
fi

# Check init container uses busybox
INIT_IMAGE=$(kubectl get pod init-wait-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers[0].image}' 2>/dev/null || echo "")
if [[ "$INIT_IMAGE" != *"busybox"* ]]; then
    echo "FAILED: Init container should use busybox image, got '$INIT_IMAGE'"
    exit 1
fi

# Check main container uses nginx
MAIN_IMAGE=$(kubectl get pod init-wait-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$MAIN_IMAGE" != *"nginx"* ]]; then
    echo "FAILED: Main container should use nginx image, got '$MAIN_IMAGE'"
    exit 1
fi

# Check pod is running (init container should have completed)
STATUS=$(kubectl get pod init-wait-pod -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

echo "SUCCESS: Init container wait pattern correctly configured"
exit 0
