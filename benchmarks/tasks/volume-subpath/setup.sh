#!/bin/bash
# Setup script for volume-subpath task

set -e

echo "Setting up volume-subpath task..."

# Create namespace if not exists
kubectl create namespace volume-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete pod subpath-pod --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete configmap app-files --namespace=volume-test --ignore-not-found=true 2>/dev/null || true

# Wait for deletion
sleep 2

echo "Setup complete."
