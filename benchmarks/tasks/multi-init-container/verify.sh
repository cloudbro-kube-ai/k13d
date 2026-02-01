#!/bin/bash
set -e

echo "Verifying multi-init-container task..."

NAMESPACE="multi-init-test"

# Check Pod exists
if ! kubectl get pod multi-init-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'multi-init-pod' not found"
    exit 1
fi

# Check all three init containers exist
INIT_CONTAINERS=$(kubectl get pod multi-init-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null || echo "")
if [[ "$INIT_CONTAINERS" != *"init-first"* ]]; then
    echo "FAILED: Init container 'init-first' not found"
    exit 1
fi
if [[ "$INIT_CONTAINERS" != *"init-second"* ]]; then
    echo "FAILED: Init container 'init-second' not found"
    exit 1
fi
if [[ "$INIT_CONTAINERS" != *"init-third"* ]]; then
    echo "FAILED: Init container 'init-third' not found"
    exit 1
fi

# Count init containers (should be 3)
INIT_COUNT=$(kubectl get pod multi-init-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers}' | grep -o '"name"' | wc -l | tr -d ' ')
if [ "$INIT_COUNT" -lt 3 ]; then
    echo "FAILED: Expected at least 3 init containers, got $INIT_COUNT"
    exit 1
fi

# Check shared volume exists
VOLUMES=$(kubectl get pod multi-init-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"shared"* ]]; then
    echo "FAILED: Volume 'shared' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/multi-init-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Check all files were created
FILES=$(kubectl exec multi-init-pod -n "$NAMESPACE" -- ls /usr/share/nginx/html/ 2>/dev/null || echo "")
if [[ "$FILES" != *"ready.txt"* ]]; then
    echo "FAILED: File 'ready.txt' not found - init containers may not have completed properly"
    exit 1
fi

echo "SUCCESS: Multi init container pattern correctly configured"
exit 0
