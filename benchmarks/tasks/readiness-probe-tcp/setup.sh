#!/bin/bash
# Setup script for readiness-probe-tcp task

set -e

echo "Setting up readiness-probe-tcp task..."

# Create namespace if not exists
kubectl create namespace probe-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing pod
kubectl delete pod readiness-tcp --namespace=probe-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
