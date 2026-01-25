#!/bin/bash
# Verifier script for create-pod task
# Checks if pod was created correctly

set -e

echo "Verifying create-pod task..."

# Check if pod exists
if ! kubectl get pod web-server --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: Pod 'web-server' not found"
    exit 1
fi

# Check if pod is running
STATUS=$(kubectl get pod web-server --namespace="${NAMESPACE}" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Check image
IMAGE=$(kubectl get pod web-server --namespace="${NAMESPACE}" -o jsonpath='{.spec.containers[0].image}')
if [[ "$IMAGE" != *"nginx"* ]]; then
    echo "ERROR: Pod is not using nginx image. Current image: $IMAGE"
    exit 1
fi

# Check labels
APP_LABEL=$(kubectl get pod web-server --namespace="${NAMESPACE}" -o jsonpath='{.metadata.labels.app}')
if [ "$APP_LABEL" != "web" ]; then
    echo "ERROR: Missing or incorrect 'app' label. Expected 'web', got '$APP_LABEL'"
    exit 1
fi

TIER_LABEL=$(kubectl get pod web-server --namespace="${NAMESPACE}" -o jsonpath='{.metadata.labels.tier}')
if [ "$TIER_LABEL" != "frontend" ]; then
    echo "ERROR: Missing or incorrect 'tier' label. Expected 'frontend', got '$TIER_LABEL'"
    exit 1
fi

# Check container port
PORT=$(kubectl get pod web-server --namespace="${NAMESPACE}" -o jsonpath='{.spec.containers[0].ports[0].containerPort}')
if [ "$PORT" != "80" ]; then
    echo "WARNING: Container port is $PORT, expected 80"
fi

echo "Verification PASSED: Pod 'web-server' created successfully with correct configuration"
exit 0
