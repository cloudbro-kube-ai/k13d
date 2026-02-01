#!/bin/bash
# Setup script for create-secret-env task

set -e

echo "Setting up create-secret-env task..."

# Cleanup any existing resources
kubectl delete pod app-pod --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete secret app-secrets --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
