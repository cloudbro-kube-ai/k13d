#!/bin/bash
# Setup script for probe-failure-threshold task

set -e

echo "Setting up probe-failure-threshold task..."

# Create namespace if not exists
kubectl create namespace probe-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing deployment
kubectl delete deployment resilient-app --namespace=probe-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
