#!/bin/bash
set -e

echo "Verifying privileged-pod task..."

NAMESPACE="privileged-test"

# Check Pod exists
if ! kubectl get pod debug-pod -n "$NAMESPACE" &>/dev/null; then
    echo "FAILED: Pod 'debug-pod' not found"
    exit 1
fi

# Check privileged is set
PRIVILEGED=$(kubectl get pod debug-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].securityContext.privileged}' 2>/dev/null || echo "")
if [[ "$PRIVILEGED" != "true" ]]; then
    echo "FAILED: privileged should be 'true', got '$PRIVILEGED'"
    exit 1
fi

# Check hostPath volume exists
VOLUME_TYPE=$(kubectl get pod debug-pod -n "$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="host-logs")].hostPath.path}' 2>/dev/null || echo "")
if [[ "$VOLUME_TYPE" != "/var/log" ]]; then
    echo "FAILED: hostPath volume should mount '/var/log', got '$VOLUME_TYPE'"
    exit 1
fi

# Check volume mount exists
MOUNT_PATH=$(kubectl get pod debug-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].volumeMounts[?(@.name=="host-logs")].mountPath}' 2>/dev/null || echo "")
if [[ "$MOUNT_PATH" != "/host-logs" ]]; then
    echo "FAILED: Volume should be mounted at '/host-logs', got '$MOUNT_PATH'"
    exit 1
fi

# Check image is busybox
IMAGE=$(kubectl get pod debug-pod -n "$NAMESPACE" -o jsonpath='{.spec.containers[0].image}' 2>/dev/null || echo "")
if [[ "$IMAGE" != *"busybox"* ]]; then
    echo "FAILED: Image should be busybox, got '$IMAGE'"
    exit 1
fi

# Check pod status
STATUS=$(kubectl get pod debug-pod -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
echo "INFO: Pod status is '$STATUS'"

echo "SUCCESS: Privileged pod correctly configured"
exit 0
