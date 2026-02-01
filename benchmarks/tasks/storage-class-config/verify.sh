#!/bin/bash
set -euo pipefail

NAMESPACE="storage-tiers"

echo "Verifying storage-class-config..."

# Check ssd-immediate StorageClass
if ! kubectl get storageclass ssd-immediate &>/dev/null; then
    echo "ERROR: StorageClass 'ssd-immediate' not found"
    exit 1
fi

SSD_BINDING=$(kubectl get storageclass ssd-immediate -o jsonpath='{.volumeBindingMode}')
if [[ "$SSD_BINDING" != "Immediate" ]]; then
    echo "ERROR: ssd-immediate volumeBindingMode should be Immediate"
    exit 1
fi

SSD_RECLAIM=$(kubectl get storageclass ssd-immediate -o jsonpath='{.reclaimPolicy}')
if [[ "$SSD_RECLAIM" != "Retain" ]]; then
    echo "ERROR: ssd-immediate reclaimPolicy should be Retain"
    exit 1
fi

SSD_EXPAND=$(kubectl get storageclass ssd-immediate -o jsonpath='{.allowVolumeExpansion}')
if [[ "$SSD_EXPAND" != "true" ]]; then
    echo "ERROR: ssd-immediate allowVolumeExpansion should be true"
    exit 1
fi

# Check hdd-topology StorageClass
if ! kubectl get storageclass hdd-topology &>/dev/null; then
    echo "ERROR: StorageClass 'hdd-topology' not found"
    exit 1
fi

HDD_BINDING=$(kubectl get storageclass hdd-topology -o jsonpath='{.volumeBindingMode}')
if [[ "$HDD_BINDING" != "WaitForFirstConsumer" ]]; then
    echo "ERROR: hdd-topology volumeBindingMode should be WaitForFirstConsumer"
    exit 1
fi

HDD_EXPAND=$(kubectl get storageclass hdd-topology -o jsonpath='{.allowVolumeExpansion}')
if [[ "$HDD_EXPAND" == "true" ]]; then
    echo "ERROR: hdd-topology allowVolumeExpansion should be false"
    exit 1
fi

# Check allowedTopologies
HDD_TOPOLOGY=$(kubectl get storageclass hdd-topology -o json | jq -r '.allowedTopologies[0].matchLabelExpressions[0].key // empty')
if [[ "$HDD_TOPOLOGY" != "topology.kubernetes.io/zone" ]]; then
    echo "ERROR: hdd-topology should have allowedTopologies with zone constraint"
    exit 1
fi

HDD_ZONES=$(kubectl get storageclass hdd-topology -o json | jq -r '.allowedTopologies[0].matchLabelExpressions[0].values | length')
if [[ "$HDD_ZONES" -lt 2 ]]; then
    echo "ERROR: hdd-topology should have at least 2 zone values"
    exit 1
fi

# Check encrypted-storage StorageClass
if ! kubectl get storageclass encrypted-storage &>/dev/null; then
    echo "ERROR: StorageClass 'encrypted-storage' not found"
    exit 1
fi

ENC_PARAM=$(kubectl get storageclass encrypted-storage -o jsonpath='{.parameters.encrypted}')
if [[ "$ENC_PARAM" != "true" ]]; then
    echo "ERROR: encrypted-storage should have parameter encrypted=true"
    exit 1
fi

ENC_TYPE=$(kubectl get storageclass encrypted-storage -o jsonpath='{.parameters.type}')
if [[ "$ENC_TYPE" != "secure" ]]; then
    echo "ERROR: encrypted-storage should have parameter type=secure"
    exit 1
fi

# Check mountOptions
ENC_MOUNT=$(kubectl get storageclass encrypted-storage -o jsonpath='{.mountOptions[*]}')
if [[ ! "$ENC_MOUNT" =~ "noatime" ]]; then
    echo "ERROR: encrypted-storage should have mountOption noatime"
    exit 1
fi

# Check namespace and PVCs
if ! kubectl get namespace $NAMESPACE &>/dev/null; then
    echo "ERROR: Namespace '$NAMESPACE' not found"
    exit 1
fi

for PVC in ssd-pvc hdd-pvc encrypted-pvc; do
    if ! kubectl get pvc $PVC -n $NAMESPACE &>/dev/null; then
        echo "ERROR: PVC '$PVC' not found in namespace '$NAMESPACE'"
        exit 1
    fi
done

# Verify PVC storage class assignments
SSD_PVC_SC=$(kubectl get pvc ssd-pvc -n $NAMESPACE -o jsonpath='{.spec.storageClassName}')
HDD_PVC_SC=$(kubectl get pvc hdd-pvc -n $NAMESPACE -o jsonpath='{.spec.storageClassName}')
ENC_PVC_SC=$(kubectl get pvc encrypted-pvc -n $NAMESPACE -o jsonpath='{.spec.storageClassName}')

if [[ "$SSD_PVC_SC" != "ssd-immediate" ]]; then
    echo "ERROR: ssd-pvc should use storageClass ssd-immediate"
    exit 1
fi

if [[ "$HDD_PVC_SC" != "hdd-topology" ]]; then
    echo "ERROR: hdd-pvc should use storageClass hdd-topology"
    exit 1
fi

if [[ "$ENC_PVC_SC" != "encrypted-storage" ]]; then
    echo "ERROR: encrypted-pvc should use storageClass encrypted-storage"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "StorageClasses are correctly configured."
exit 0
