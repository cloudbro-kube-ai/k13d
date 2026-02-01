#!/bin/bash
set -e

echo "Setting up cross-namespace-rbac task..."

# Clean up any existing resources
kubectl delete namespace monitoring --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace app-frontend --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace app-backend --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrole pod-reader --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete. Create the cross-namespace RBAC configuration."
