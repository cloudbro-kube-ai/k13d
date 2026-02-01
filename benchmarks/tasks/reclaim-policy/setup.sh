#!/bin/bash
set -e

NAMESPACE="reclaim-demo"

echo "Setting up reclaim-policy task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete pv pv-retain pv-delete pv-recycle --ignore-not-found=true 2>/dev/null || true

# Clean up host paths
rm -rf /tmp/pv-retain /tmp/pv-delete /tmp/pv-recycle 2>/dev/null || true
mkdir -p /tmp/pv-retain /tmp/pv-delete /tmp/pv-recycle

sleep 2

echo "Setup complete. Create PVs with different reclaim policies."
