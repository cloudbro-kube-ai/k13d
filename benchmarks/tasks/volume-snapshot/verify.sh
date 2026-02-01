#!/bin/bash
set -euo pipefail

NAMESPACE="snapshot-demo"

echo "Verifying volume-snapshot..."

# Check VolumeSnapshotClass exists (might need CRDs)
if kubectl api-resources | grep -q volumesnapshotclasses; then
    if ! kubectl get volumesnapshotclass csi-snapshot-class &>/dev/null; then
        echo "ERROR: VolumeSnapshotClass 'csi-snapshot-class' not found"
        exit 1
    fi

    # Verify VolumeSnapshotClass settings
    VSC_POLICY=$(kubectl get volumesnapshotclass csi-snapshot-class -o jsonpath='{.deletionPolicy}')
    if [[ "$VSC_POLICY" != "Delete" ]]; then
        echo "ERROR: VolumeSnapshotClass deletionPolicy should be Delete"
        exit 1
    fi

    VSC_DEFAULT=$(kubectl get volumesnapshotclass csi-snapshot-class -o jsonpath='{.metadata.annotations.snapshot\.storage\.kubernetes\.io/is-default-class}')
    if [[ "$VSC_DEFAULT" != "true" ]]; then
        echo "WARNING: VolumeSnapshotClass should have default annotation"
    fi
else
    echo "WARNING: VolumeSnapshot CRDs not installed in cluster, skipping VSC checks"
fi

# Check namespace exists
if ! kubectl get namespace $NAMESPACE &>/dev/null; then
    echo "ERROR: Namespace '$NAMESPACE' not found"
    exit 1
fi

# Check source PVC exists
if ! kubectl get pvc source-data -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PVC 'source-data' not found"
    exit 1
fi

# Check data-writer pod exists
if ! kubectl get pod data-writer -n $NAMESPACE &>/dev/null; then
    echo "ERROR: Pod 'data-writer' not found"
    exit 1
fi

# Check VolumeSnapshot exists
if kubectl api-resources | grep -q volumesnapshots; then
    if ! kubectl get volumesnapshot source-data-snap -n $NAMESPACE &>/dev/null; then
        echo "ERROR: VolumeSnapshot 'source-data-snap' not found"
        exit 1
    fi

    # Verify snapshot source
    SNAP_SOURCE=$(kubectl get volumesnapshot source-data-snap -n $NAMESPACE -o jsonpath='{.spec.source.persistentVolumeClaimName}')
    if [[ "$SNAP_SOURCE" != "source-data" ]]; then
        echo "ERROR: Snapshot source should be 'source-data', got '$SNAP_SOURCE'"
        exit 1
    fi

    SNAP_CLASS=$(kubectl get volumesnapshot source-data-snap -n $NAMESPACE -o jsonpath='{.spec.volumeSnapshotClassName}')
    if [[ "$SNAP_CLASS" != "csi-snapshot-class" ]]; then
        echo "ERROR: Snapshot class should be 'csi-snapshot-class', got '$SNAP_CLASS'"
        exit 1
    fi
else
    echo "WARNING: VolumeSnapshot CRDs not installed, skipping snapshot checks"
fi

# Check restored PVC exists
if ! kubectl get pvc restored-data -n $NAMESPACE &>/dev/null; then
    echo "ERROR: PVC 'restored-data' not found"
    exit 1
fi

# Verify restored PVC dataSource
RESTORED_SOURCE_KIND=$(kubectl get pvc restored-data -n $NAMESPACE -o jsonpath='{.spec.dataSource.kind}')
RESTORED_SOURCE_NAME=$(kubectl get pvc restored-data -n $NAMESPACE -o jsonpath='{.spec.dataSource.name}')

if [[ "$RESTORED_SOURCE_KIND" != "VolumeSnapshot" ]]; then
    echo "ERROR: restored-data dataSource.kind should be VolumeSnapshot, got '$RESTORED_SOURCE_KIND'"
    exit 1
fi

if [[ "$RESTORED_SOURCE_NAME" != "source-data-snap" ]]; then
    echo "ERROR: restored-data dataSource.name should be 'source-data-snap', got '$RESTORED_SOURCE_NAME'"
    exit 1
fi

echo "--- Verification Successful! ---"
echo "Volume snapshot configuration is correct."
exit 0
