#!/bin/bash
set -e

echo "Verifying init-container-setup task..."

NAMESPACE="init-setup-test"

# Check Pod exists
if ! kubectl get pod setup-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'setup-pod' not found"
    exit 1
fi

# Check init container exists with correct name
INIT_CONTAINERS=$(kubectl get pod setup-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null || echo "")
if [[ "$INIT_CONTAINERS" != *"setup"* ]]; then
    echo "FAILED: Init container 'setup' not found"
    exit 1
fi

# Check emptyDir volume exists
VOLUMES=$(kubectl get pod setup-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"workdir"* ]]; then
    echo "FAILED: Volume 'workdir' not found"
    exit 1
fi

# Check pod is running
STATUS=$(kubectl get pod setup-pod -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

# Wait for pod to be ready
kubectl wait --for=condition=Ready pod/setup-pod -n "$NAMESPACE" --timeout=60s

# Check the content served by nginx
CONTENT=$(kubectl exec setup-pod -n "$NAMESPACE" -- cat /usr/share/nginx/html/index.html 2>/dev/null || echo "")
if [[ "$CONTENT" != *"Hello from init container"* ]]; then
    echo "FAILED: Content not properly set up by init container, got '$CONTENT'"
    exit 1
fi

echo "SUCCESS: Init container setup pattern correctly configured"
exit 0
