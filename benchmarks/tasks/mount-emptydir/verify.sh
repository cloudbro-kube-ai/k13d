#!/bin/bash
# Verifier script for mount-emptydir task

set -e

echo "Verifying mount-emptydir task..."

NAMESPACE="volume-test"

# Check if pod exists
if ! kubectl get pod shared-data --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: Pod 'shared-data' not found"
    exit 1
fi

# Check if pod is running
STATUS=$(kubectl get pod shared-data --namespace="$NAMESPACE" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Running" ]; then
    echo "ERROR: Pod is not running. Current status: $STATUS"
    exit 1
fi

# Check for two containers
CONTAINER_COUNT=$(kubectl get pod shared-data --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' | wc -w)
if [ "$CONTAINER_COUNT" -lt 2 ]; then
    echo "ERROR: Pod should have at least 2 containers, found $CONTAINER_COUNT"
    exit 1
fi

# Check for emptyDir volume
VOLUME_TYPE=$(kubectl get pod shared-data --namespace="$NAMESPACE" -o jsonpath='{.spec.volumes[?(@.name=="shared-volume")].emptyDir}')
if [ -z "$VOLUME_TYPE" ]; then
    echo "ERROR: emptyDir volume 'shared-volume' not found"
    exit 1
fi

# Check volume mounts in writer container
WRITER_MOUNT=$(kubectl get pod shared-data --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[?(@.name=="writer")].volumeMounts[?(@.name=="shared-volume")].mountPath}')
if [ "$WRITER_MOUNT" != "/data" ]; then
    echo "ERROR: Writer container volume mount not found at /data"
    exit 1
fi

# Check volume mounts in reader container
READER_MOUNT=$(kubectl get pod shared-data --namespace="$NAMESPACE" -o jsonpath='{.spec.containers[?(@.name=="reader")].volumeMounts[?(@.name=="shared-volume")].mountPath}')
if [ "$READER_MOUNT" != "/data" ]; then
    echo "ERROR: Reader container volume mount not found at /data"
    exit 1
fi

echo "Verification PASSED: Pod 'shared-data' created with emptyDir volume shared between containers"
exit 0
