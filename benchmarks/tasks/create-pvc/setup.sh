#!/bin/bash
# Setup script for create-pvc task

set -e

echo "Setting up create-pvc task..."

# Delete any existing PVC with the same name
kubectl delete pvc data-pvc --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

# Wait for PVC to be deleted
sleep 2

echo "Setup complete."
