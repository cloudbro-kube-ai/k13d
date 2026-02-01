#!/bin/bash
set -e

NAMESPACE="snapshot-demo"

echo "Setting up volume-snapshot task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete volumesnapshotclass csi-snapshot-class --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete. Create VolumeSnapshotClass and snapshots."
echo "NOTE: Volume snapshots require CSI driver support. Resources may stay pending without CSI."
