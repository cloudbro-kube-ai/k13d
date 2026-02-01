#!/bin/bash
set -e

echo "Verifying read-only-rootfs task..."

NAMESPACE="readonly-fs-test"

# Check Pod exists
if ! kubectl get pod readonly-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'readonly-pod' not found"
    exit 1
fi

# Check readOnlyRootFilesystem is set
READONLY=$(kubectl get pod readonly-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.readOnlyRootFilesystem}' 2>/dev/null || echo "")
if [[ "$READONLY" != "true" ]]; then
    echo "FAILED: readOnlyRootFilesystem should be 'true', got '$READONLY'"
    exit 1
fi

# Check volumes exist
VOLUMES=$(kubectl get pod readonly-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"cache"* ]]; then
    echo "FAILED: Volume 'cache' not found"
    exit 1
fi
if [[ "$VOLUMES" != *"run"* ]]; then
    echo "FAILED: Volume 'run' not found"
    exit 1
fi
if [[ "$VOLUMES" != *"tmp"* ]]; then
    echo "FAILED: Volume 'tmp' not found"
    exit 1
fi

# Check volume mounts
MOUNTS=$(kubectl get pod readonly-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[*].mountPath}' 2>/dev/null || echo "")
if [[ "$MOUNTS" != *"/var/cache/nginx"* ]]; then
    echo "FAILED: Volume mount for '/var/cache/nginx' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/readonly-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

echo "SUCCESS: Read-only root filesystem correctly configured"
exit 0
