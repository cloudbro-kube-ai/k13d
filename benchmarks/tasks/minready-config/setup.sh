#!/bin/bash
# Setup script for minready-config task

set -e

echo "Setting up minready-config task..."

# Create namespace if not exists
kubectl create namespace deploy-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing deployment
kubectl delete deployment stable-app --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
