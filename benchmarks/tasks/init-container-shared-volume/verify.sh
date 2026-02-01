#!/bin/bash
set -e

echo "Verifying init-container-shared-volume task..."

NAMESPACE="init-volume-test"

# Check Pod exists
if ! kubectl get pod download-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'download-pod' not found"
    exit 1
fi

# Check init container exists
INIT_CONTAINERS=$(kubectl get pod download-pod -n "$NAMESPACE" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null || echo "")
if [[ "$INIT_CONTAINERS" != *"downloader"* ]]; then
    echo "FAILED: Init container 'downloader' not found"
    exit 1
fi

# Check volume exists
VOLUMES=$(kubectl get pod download-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"data-volume"* ]]; then
    echo "FAILED: Volume 'data-volume' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/download-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Check index.html was created
INDEX_CONTENT=$(kubectl exec download-pod -n "$NAMESPACE" -- cat /usr/share/nginx/html/index.html 2>/dev/null || echo "")
if [[ "$INDEX_CONTENT" != *"Downloaded by init container"* ]]; then
    echo "FAILED: index.html content not correct"
    exit 1
fi

# Check config.json was created
CONFIG_CONTENT=$(kubectl exec download-pod -n "$NAMESPACE" -- cat /usr/share/nginx/html/config.json 2>/dev/null || echo "")
if [[ "$CONFIG_CONTENT" != *"initialized"* ]]; then
    echo "FAILED: config.json not found or incorrect"
    exit 1
fi

echo "SUCCESS: Init container shared volume pattern correctly configured"
exit 0
