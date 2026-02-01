#!/bin/bash
# Setup script for external-name-service task

set -e

echo "Setting up external-name-service task..."

# Create namespace if not exists
kubectl create namespace service-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing service
kubectl delete service external-db --namespace=service-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
