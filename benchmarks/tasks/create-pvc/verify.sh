#!/bin/bash
# Verifier script for create-pvc task

set -e

echo "Verifying create-pvc task..."

# Check if PVC exists
if ! kubectl get pvc data-pvc --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: PVC 'data-pvc' not found"
    exit 1
fi

# Check PVC status (Bound or Pending are acceptable)
STATUS=$(kubectl get pvc data-pvc --namespace="${NAMESPACE}" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Bound" ] && [ "$STATUS" != "Pending" ]; then
    echo "ERROR: PVC status is '$STATUS', expected 'Bound' or 'Pending'"
    exit 1
fi

# Check access mode
ACCESS_MODE=$(kubectl get pvc data-pvc --namespace="${NAMESPACE}" -o jsonpath='{.spec.accessModes[0]}')
if [ "$ACCESS_MODE" != "ReadWriteOnce" ]; then
    echo "ERROR: Access mode is '$ACCESS_MODE', expected 'ReadWriteOnce'"
    exit 1
fi

# Check storage request
STORAGE=$(kubectl get pvc data-pvc --namespace="${NAMESPACE}" -o jsonpath='{.spec.resources.requests.storage}')
if [ "$STORAGE" != "1Gi" ]; then
    echo "WARNING: Storage request is '$STORAGE', expected '1Gi'"
fi

# Check labels
APP_LABEL=$(kubectl get pvc data-pvc --namespace="${NAMESPACE}" -o jsonpath='{.metadata.labels.app}')
if [ "$APP_LABEL" != "data" ]; then
    echo "ERROR: Missing or incorrect 'app' label. Expected 'data', got '$APP_LABEL'"
    exit 1
fi

echo "Verification PASSED: PVC 'data-pvc' created successfully"
exit 0
