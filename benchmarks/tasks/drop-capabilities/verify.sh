#!/bin/bash
set -e

echo "Verifying drop-capabilities task..."

NAMESPACE="drop-cap-test"

# Check Pod exists
if ! kubectl get pod secure-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'secure-pod' not found"
    exit 1
fi

# Check capabilities are dropped
DROP_CAPS=$(kubectl get pod secure-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.capabilities.drop}' 2>/dev/null || echo "")
if [[ "$DROP_CAPS" != *"ALL"* ]]; then
    echo "FAILED: Should drop ALL capabilities, got '$DROP_CAPS'"
    exit 1
fi

# Check allowPrivilegeEscalation is false
ALLOW_PRIV_ESC=$(kubectl get pod secure-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.allowPrivilegeEscalation}' 2>/dev/null || echo "")
if [[ "$ALLOW_PRIV_ESC" != "false" ]]; then
    echo "FAILED: allowPrivilegeEscalation should be 'false', got '$ALLOW_PRIV_ESC'"
    exit 1
fi

# Check image is alpine
IMAGE=$(kubectl get pod secure-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$IMAGE" != *"alpine"* ]]; then
    echo "FAILED: Image should be alpine, got '$IMAGE'"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/secure-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

echo "SUCCESS: Dropped capabilities correctly configured"
exit 0
