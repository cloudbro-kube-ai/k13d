#!/bin/bash
# Setup script for exec-probe task

set -e

echo "Setting up exec-probe task..."

# Create namespace if not exists
kubectl create namespace probe-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing pod
kubectl delete pod exec-probe-pod --namespace=probe-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
