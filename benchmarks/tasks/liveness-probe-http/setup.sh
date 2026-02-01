#!/bin/bash
# Setup script for liveness-probe-http task

set -e

echo "Setting up liveness-probe-http task..."

# Create namespace if not exists
kubectl create namespace probe-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing pod
kubectl delete pod liveness-http --namespace=probe-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
