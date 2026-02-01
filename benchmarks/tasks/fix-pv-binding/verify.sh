#!/bin/bash
# Verifier script for fix-pv-binding task

set -e

echo "Verifying fix-pv-binding task..."

# Check if PVC exists
if ! kubectl get pvc app-data --namespace="${NAMESPACE}" &>/dev/null; then
    echo "ERROR: PVC 'app-data' not found"
    exit 1
fi

# Check PVC status - must be Bound
STATUS=$(kubectl get pvc app-data --namespace="${NAMESPACE}" -o jsonpath='{.status.phase}')
if [ "$STATUS" != "Bound" ]; then
    echo "ERROR: PVC is not Bound. Current status: $STATUS"
    echo "The task requires fixing the binding issue so PVC becomes Bound."
    exit 1
fi

# Verify it's bound to our PV or a dynamically provisioned one
VOLUME_NAME=$(kubectl get pvc app-data --namespace="${NAMESPACE}" -o jsonpath='{.spec.volumeName}')
if [ -z "$VOLUME_NAME" ]; then
    echo "ERROR: PVC is not bound to any volume"
    exit 1
fi

echo "Verification PASSED: PVC 'app-data' is successfully Bound to volume '$VOLUME_NAME'"
exit 0
