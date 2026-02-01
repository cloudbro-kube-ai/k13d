#!/bin/bash
set -e

NAMESPACE="templating-demo"

echo "Setting up helm-like-templating task..."

# Clean up any existing resources
kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true

sleep 2

# Create namespace
kubectl create namespace $NAMESPACE

echo "Setup complete. Create the ConfigMap-based templating system."
