#!/bin/bash
# Verifier script for mount-hostpath task

set -e

echo "Verifying mount-hostpath task..."

NAMESPACE="volume-test"

# Check if pod exists
if ! kubectl get pod hostpath-pod --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'hostpath-pod' not found"
    exit 1
fi

# Check if pod is running
STATUS=$(kubectl get pod hostpath-pod --namespace="$NAMESPACE" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Check for hostPath volume
HOST_PATH=$(kubectl get pod hostpath-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="host-volume")].hostPath.path}')
if [ -z "$HOST_PATH" ]; then
    echo "ERROR: hostPath volume 'host-volume' not found"
    exit 1
fi

if [ "$HOST_PATH" != "/tmp/k8s-logs" ]; then
    echo "ERROR: hostPath should be /tmp/k8s-logs, got $HOST_PATH"
    exit 1
fi

# Check hostPath type
HOST_TYPE=$(kubectl get pod hostpath-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="host-volume")].hostPath.type}')
if [ "$HOST_TYPE" != "DirectoryOrCreate" ]; then
    echo "ERROR: hostPath type should be DirectoryOrCreate, got $HOST_TYPE"
    exit 1
fi

# Check volume mount
MOUNT_PATH=$(kubectl get pod hostpath-pod --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[?(@.name=="logger")].volumeMounts[?(@.name=="host-volume")].mountPath}')
if [ "$MOUNT_PATH" != "/host-logs" ]; then
    echo "ERROR: Volume mount path should be /host-logs, got $MOUNT_PATH"
    exit 1
fi

echo "Verification PASSED: Pod 'hostpath-pod' created with hostPath volume correctly configured"
exit 0
