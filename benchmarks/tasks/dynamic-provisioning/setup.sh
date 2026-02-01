#!/bin/bash
set -e

NAMESPACE="storage-demo"

echo "Setting up dynamic-provisioning task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete storageclass fast-storage --ignore-not-found=true 2>/dev/null || true
kubectl delete pv fast-pv-1 --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete. Create the StorageClass and PVC for dynamic provisioning."
