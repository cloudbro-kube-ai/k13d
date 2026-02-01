#!/bin/bash
set -euo pipefail

NAMESPACE="reclaim-demo"
TIMEOUT="120s"

echo "Verifying reclaim-policy..."

# Check namespace exists
if ! kubectl get namespace $NAMESPACE &>/dev/null; then
    echo "ERROR: Namespace '$NAMESPACE' not found"
    exit 1
fi

# Check PVs exist with correct reclaim policies
if ! kubectl get pv pv-retain &>/dev/null; then
    echo "ERROR: PV 'pv-retain' not found"
    exit 1
fi

RETAIN_POLICY=$(kubectl get pv pv-retain -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')
if [[ "$RETAIN_POLICY" != "Retain" ]]; then
    echo "ERROR: pv-retain should have reclaimPolicy Retain, got '$RETAIN_POLICY'"
    exit 1
fi

if ! kubectl get pv pv-delete &>/dev/null; then
    echo "ERROR: PV 'pv-delete' not found"
    exit 1
fi

# pv-delete should now have Retain policy (after patch)
DELETE_POLICY=$(kubectl get pv pv-delete -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')
if [[ "$DELETE_POLICY" != "Retain" ]]; then
    echo "ERROR: pv-delete should be patched to Retain, got '$DELETE_POLICY'"
    exit 1
fi

if ! kubectl get pv pv-recycle &>/dev/null; then
    echo "ERROR: PV 'pv-recycle' not found"
    exit 1
fi

RECYCLE_POLICY=$(kubectl get pv pv-recycle -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')
if [[ "$RECYCLE_POLICY" != "Recycle" ]]; then
    echo "WARNING: pv-recycle policy is '$RECYCLE_POLICY' (Recycle is deprecated in newer K8s)"
fi

# Check storageClassName
for PV in pv-retain pv-delete pv-recycle; do
    SC=$(kubectl get pv $PV -o jsonpath='{.spec.storageClassName}')
    if [[ "$SC" != "manual" ]]; then
        echo "ERROR: $PV should have storageClassName 'manual'"
        exit 1
    fi
done

# Check PVCs exist
for PVC in pvc-retain pvc-delete pvc-recycle; do
    if ! kubectl get pvc $PVC -n $NAMESPACE &>/dev/null; then
        echo "ERROR: PVC '$PVC' not found"
        exit 1
    fi
done

# Check PVC bindings
RETAIN_BOUND=$(kubectl get pvc pvc-retain -n $NAMESPACE -o jsonpath='{.spec.volumeName}')
if [[ "$RETAIN_BOUND" != "pv-retain" ]]; then
    echo "ERROR: pvc-retain should be bound to pv-retain"
    exit 1
fi

DELETE_BOUND=$(kubectl get pvc pvc-delete -n $NAMESPACE -o jsonpath='{.spec.volumeName}')
if [[ "$DELETE_BOUND" != "pv-delete" ]]; then
    echo "ERROR: pvc-delete should be bound to pv-delete"
    exit 1
fi

RECYCLE_BOUND=$(kubectl get pvc pvc-recycle -n $NAMESPACE -o jsonpath='{.spec.volumeName}')
if [[ "$RECYCLE_BOUND" != "pv-recycle" ]]; then
    echo "ERROR: pvc-recycle should be bound to pv-recycle"
    exit 1
fi

# Check Pod exists and mounts all PVCs
if ! kubectl get pod policy-demo -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Pod 'policy-demo' not found"
    exit 1
fi

POD_VOLUMES=$(kubectl get pod policy-demo -n $NAMESPACE -o json | jq -r '.spec.volumes[].persistentVolumeClaim.claimName // empty' | sort | tr '\n' ' ')
for PVC in pvc-delete pvc-recycle pvc-retain; do
    if [[ ! "$POD_VOLUMES" =~ "$PVC" ]]; then
        echo "ERROR: Pod doesn't mount PVC '$PVC'"
        exit 1
    fi
done

echo "--- Verification Successful! ---"
echo "PV reclaim policies are correctly configured."
exit 0
