#!/bin/bash
# Setup script for node-affinity task

set -e

echo "Setting up node-affinity task..."

# Create namespace if not exists
kubectl create namespace schedule-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing pod
kubectl delete pod affinity-pod --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
