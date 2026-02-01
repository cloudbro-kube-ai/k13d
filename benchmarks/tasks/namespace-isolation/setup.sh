#!/bin/bash
set -e

echo "Setting up namespace-isolation task..."

# Clean up any existing resources
for NS in tenant-alpha tenant-beta tenant-gamma shared-services; do
    kubectl delete namespace $NS --ignore-not-found=true 2>/dev/null || true
done

sleep 2

echo "Setup complete. Create the multi-tenant isolation policies."
