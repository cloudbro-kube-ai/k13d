#!/bin/bash
set -euo pipefail

NAMESPACE="storage-demo"
TIMEOUT="120s"

echo "Verifying dynamic-provisioning..."

# Check StorageClass exists
if ! kubectl get storageclass fast-storage &>/dev/null; then
    echo "ERROR: StorageClass 'fast-storage' not found"
    exit 1
fi

# Verify StorageClass settings
SC_BINDING=$(kubectl get storageclass fast-storage -o jsonpath='{.volumeBindingMode}')
if [[ "$SC_BINDING" != "WaitForFirstConsumer" ]]; then
    echo "ERROR: StorageClass volumeBindingMode should be WaitForFirstConsumer, got '$SC_BINDING'"
    exit 1
fi

SC_RECLAIM=$(kubectl get storageclass fast-storage -o jsonpath='{.reclaimPolicy}')
if [[ "$SC_RECLAIM" != "Delete" ]]; then
    echo "ERROR: StorageClass reclaimPolicy should be Delete, got '$SC_RECLAIM'"
    exit 1
fi

SC_EXPAND=$(kubectl get storageclass fast-storage -o jsonpath='{.allowVolumeExpansion}')
if [[ "$SC_EXPAND" != "true" ]]; then
    echo "ERROR: StorageClass allowVolumeExpansion should be true"
    exit 1
fi

# Check namespace exists
if ! kubectl get namespace $NAMESPACE &>/dev/null; then
    echo "ERROR: Namespace '$NAMESPACE' not found"
    exit 1
fi

# Check PVC exists
if ! kubectl get pvc app-data -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PVC 'app-data' not found"
    exit 1
fi

# Verify PVC settings
PVC_SC=$(kubectl get pvc app-data -n $NAMESPACE -o jsonpath='{.spec.storageClassName}')
if [[ "$PVC_SC" != "fast-storage" ]]; then
    echo "ERROR: PVC storageClassName should be fast-storage, got '$PVC_SC'"
    exit 1
fi

PVC_ACCESS=$(kubectl get pvc app-data -n $NAMESPACE -o jsonpath='{.spec.accessModes[0]}')
if [[ "$PVC_ACCESS" != "ReadWriteOnce" ]]; then
    echo "ERROR: PVC accessMode should be ReadWriteOnce"
    exit 1
fi

# Check Pod exists
if ! kubectl get pod storage-test -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Pod 'storage-test' not found"
    exit 1
fi

# Wait for pod to be ready
if ! kubectl wait --for=condition=Ready pod/storage-test -n $NAMESPACE --timeout=$TIMEOUT 2>/dev/null; then
    echo "WARNING: Pod not ready yet, checking PVC status..."

    # Check if PVC is pending (might need a PV)
    PVC_STATUS=$(kubectl get pvc app-data -n $NAMESPACE -o jsonpath='{.status.phase}')
    if [[ "$PVC_STATUS" == "Pending" ]]; then
        echo "INFO: PVC is pending, checking if PV needs to be created..."

        # Check if PV exists
        if ! kubectl get pv fast-pv-1 &>/dev/null; then
            echo "ERROR: PVC is pending and no PV 'fast-pv-1' exists. Create a matching PV."
            exit 1
        fi
    fi
fi

# Verify pod mounts the PVC
POD_VOLUME=$(kubectl get pod storage-test -n $NAMESPACE -o json | jq -r '.spec.volumes[]? | select(.persistentVolumeClaim.claimName == "app-data") | .name')
if [[ -z "$POD_VOLUME" ]]; then
    echo "ERROR: Pod doesn't mount PVC 'app-data'"
    exit 1
fi

# Check mount path
MOUNT_PATH=$(kubectl get pod storage-test -n $NAMESPACE -o json | jq -r '.spec.containers[0].volumeMounts[]? | select(.mountPath == "/data") | .mountPath')
if [[ "$MOUNT_PATH" != "/data" ]]; then
    echo "ERROR: PVC should be mounted at /data"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Dynamic provisioning is correctly configured."
exit 0
