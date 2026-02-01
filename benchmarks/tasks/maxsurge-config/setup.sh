#!/bin/bash
# Setup script for maxsurge-config task

set -e

echo "Setting up maxsurge-config task..."

# Create namespace if not exists
kubectl create namespace deploy-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing deployment
kubectl delete deployment web-surge --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
