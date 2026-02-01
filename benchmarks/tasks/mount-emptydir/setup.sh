#!/bin/bash
# Setup script for mount-emptydir task

set -e

echo "Setting up mount-emptydir task..."

# Create namespace if not exists
kubectl create namespace volume-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing pod with the same name
kubectl delete pod shared-data --namespace=volume-test --ignore-not-found=true 2>/dev/null || true

# Wait for pod to be deleted
sleep 2

echo "Setup complete."
