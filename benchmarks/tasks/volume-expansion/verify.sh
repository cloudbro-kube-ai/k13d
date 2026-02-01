#!/bin/bash
set -euo pipefail

NAMESPACE="expansion-demo"

echo "Verifying volume-expansion..."

# Check StorageClass has allowVolumeExpansion enabled
SC_EXPAND=$(kubectl get storageclass expandable-sc -o jsonpath='{.allowVolumeExpansion}')
if [[ "$SC_EXPAND" != "true" ]]; then
    echo "ERROR: StorageClass 'expandable-sc' should have allowVolumeExpansion=true"
    exit 1
fi

# Check app-storage PVC exists
if ! kubectl get pvc app-storage -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PVC 'app-storage' not found"
    exit 1
fi

# Verify app-storage has been expanded to 5Gi
APP_STORAGE_REQUEST=$(kubectl get pvc app-storage -n $NAMESPACE -o jsonpath='{.spec.resources.requests.storage}')
if [[ "$APP_STORAGE_REQUEST" != "5Gi" ]]; then
    echo "ERROR: app-storage should be expanded to 5Gi, got '$APP_STORAGE_REQUEST'"
    exit 1
fi

# Check backup-storage PVC exists
if ! kubectl get pvc backup-storage -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PVC 'backup-storage' not found"
    exit 1
fi

# Verify backup-storage has been expanded to 4Gi
BACKUP_STORAGE_REQUEST=$(kubectl get pvc backup-storage -n $NAMESPACE -o jsonpath='{.spec.resources.requests.storage}')
if [[ "$BACKUP_STORAGE_REQUEST" != "4Gi" ]]; then
    echo "ERROR: backup-storage should be expanded to 4Gi, got '$BACKUP_STORAGE_REQUEST'"
    exit 1
fi

# Check that backup-storage uses the expandable-sc StorageClass
BACKUP_SC=$(kubectl get pvc backup-storage -n $NAMESPACE -o jsonpath='{.spec.storageClassName}')
if [[ "$BACKUP_SC" != "expandable-sc" ]]; then
    echo "ERROR: backup-storage should use StorageClass 'expandable-sc'"
    exit 1
fi

# Verify StatefulSet is still running (online expansion)
STS_READY=$(kubectl get statefulset data-app -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
if [[ "$STS_READY" != "1" ]]; then
    echo "ERROR: StatefulSet should still be running during/after expansion"
    exit 1
fi

# Check pod is ready
POD_READY=$(kubectl get pods -l app=data-app -n $NAMESPACE -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}')
if [[ "$POD_READY" != "True" ]]; then
    echo "ERROR: Pod should be ready (online expansion)"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "PVC volume expansion completed correctly."
exit 0
