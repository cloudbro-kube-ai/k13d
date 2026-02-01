#!/bin/bash
# Verifier script for pvc-resize task

set -e

echo "Verifying pvc-resize task..."

NAMESPACE="volume-test"

# Check if PVC exists
if ! kubectl get pvc data-pvc --namespace="$NAMESPACE" &>/dev/null; then
    echo "ERROR: PVC 'data-pvc' not found"
    exit 1
fi

# Check the requested storage size
REQUESTED_SIZE=$(kubectl get pvc data-pvc --namespace="$NAMESPACE" -o jsonpath='{.spec.resources.requests.storage}')

# Convert to number for comparison (handle both Gi and G notation)
SIZE_NUM=$(echo "$REQUESTED_SIZE" | sed 's/[^0-9]//g')

if [ "$SIZE_NUM" -lt 2 ]; then
    echo "ERROR: PVC requested size should be at least 2Gi, got $REQUESTED_SIZE"
    exit 1
fi

echo "Verification PASSED: PVC 'data-pvc' resize request to $REQUESTED_SIZE has been accepted"
exit 0
