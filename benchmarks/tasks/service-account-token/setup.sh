#!/bin/bash
set -e

NAMESPACE="token-demo"

echo "Setting up service-account-token task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

echo "Setup complete. Configure bound service account tokens with custom settings."
