#!/bin/bash
set -e

NAMESPACE="tenant-platform"

echo "Setting up rbac-impersonation task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrole impersonation-role --ignore-not-found=true 2>/dev/null || true
kubectl delete clusterrolebinding platform-impersonation --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

echo "Setup complete. Configure RBAC impersonation for the multi-tenant platform."
