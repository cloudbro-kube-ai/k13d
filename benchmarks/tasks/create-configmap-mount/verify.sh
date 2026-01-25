#!/bin/bash
set -e

echo "Verifying create-configmap-mount task..."

# Check ConfigMap exists
if ! kubectl get configmap app-config -n config-test &>/dev/null; then
    echo "FAILED: ConfigMap 'app-config' not found"
    exit 1
fi

# Check ConfigMap has the correct data
DATA=$(kubectl get configmap app-config -n config-test -o jsonpath='{.data.app\.properties}' 2>/dev/null || echo "")
if [[ "$DATA" != *"debug=true"* ]]; then
    echo "FAILED: ConfigMap data incorrect, expected 'debug=true', got '$DATA'"
    exit 1
fi

# Check Pod exists and is running
STATUS=$(kubectl get pod config-pod -n config-test -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$STATUS" != "Running" ]; then
    echo "FAILED: Pod not in Running state, got '$STATUS'"
    exit 1
fi

# Check volume mount exists at /etc/config
MOUNT_PATH=$(kubectl get pod config-pod -n config-test -o jsonpath='{.spec.containers[0].volumeMounts[*].mountPath}' 2>/dev/null || echo "")
if [[ "$MOUNT_PATH" != *"/etc/config"* ]]; then
    echo "FAILED: Volume not mounted at /etc/config"
    exit 1
fi

echo "SUCCESS: ConfigMap and Pod correctly configured"
exit 0
