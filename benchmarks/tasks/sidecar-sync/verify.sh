#!/bin/bash
set -e

echo "Verifying sidecar-sync task..."

NAMESPACE="sidecar-sync-test"

# Check Pod exists
if ! kubectl get pod sync-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'sync-pod' not found"
    exit 1
fi

# Check both containers exist
CONTAINERS=$(kubectl get pod sync-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")
if [[ "$CONTAINERS" != *"web"* ]]; then
    echo "FAILED: Container 'web' not found"
    exit 1
fi
if [[ "$CONTAINERS" != *"content-sync"* ]]; then
    echo "FAILED: Container 'content-sync' not found"
    exit 1
fi

# Check shared volume exists
VOLUMES=$(kubectl get pod sync-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"content"* ]]; then
    echo "FAILED: Volume 'content' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/sync-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Wait for content to be synced
sleep 10

# Check content was synced
CONTENT=$(kubectl exec sync-pod -n "$NAMESPACE" -c web -- cat /usr/share/nginx/html/index.html 2>/dev/null || echo "")
if [[ -z "$CONTENT" ]]; then
    echo "FAILED: Content not synced - index.html is empty or missing"
    exit 1
fi

if [[ "$CONTENT" != *"synced"* ]]; then
    echo "WARNING: Content may not be from sync sidecar"
fi

echo "SUCCESS: Sidecar sync pattern correctly configured"
exit 0
