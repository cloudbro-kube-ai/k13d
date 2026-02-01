#!/bin/bash
set -e

echo "Verifying sidecar-logging task..."

NAMESPACE="sidecar-log-test"

# Check Pod exists
if ! kubectl get pod logging-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'logging-pod' not found"
    exit 1
fi

# Check both containers exist
CONTAINERS=$(kubectl get pod logging-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")
if [[ "$CONTAINERS" != *"app"* ]]; then
    echo "FAILED: Container 'app' not found"
    exit 1
fi
if [[ "$CONTAINERS" != *"log-shipper"* ]]; then
    echo "FAILED: Container 'log-shipper' not found"
    exit 1
fi

# Check shared volume exists
VOLUMES=$(kubectl get pod logging-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[*].name}' 2>/dev/null || echo "")
if [[ "$VOLUMES" != *"log-volume"* ]]; then
    echo "FAILED: Volume 'log-volume' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/logging-pod -n "$NAMESPACE" --timeout=120s; then
    echo "FAILED: Pod did not become ready in time"
    exit 1
fi

# Wait a bit for logs to be generated
sleep 10

# Check that logs are being written
LOG_CONTENT=$(kubectl exec logging-pod -n "$NAMESPACE" -c app -- cat /var/log/app.log 2>/dev/null || echo "")
if [[ -z "$LOG_CONTENT" ]]; then
    echo "FAILED: No logs found in /var/log/app.log"
    exit 1
fi

# Check sidecar can read the logs
SIDECAR_ACCESS=$(kubectl exec logging-pod -n "$NAMESPACE" -c log-shipper -- cat /var/log/app.log 2>/dev/null || echo "")
if [[ -z "$SIDECAR_ACCESS" ]]; then
    echo "FAILED: Sidecar cannot access log file"
    exit 1
fi

echo "SUCCESS: Sidecar logging pattern correctly configured"
exit 0
