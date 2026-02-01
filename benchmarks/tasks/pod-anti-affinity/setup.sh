#!/bin/bash
# Setup script for pod-anti-affinity task

set -e

echo "Setting up pod-anti-affinity task..."

# Create namespace if not exists
kubectl create namespace schedule-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing deployment
kubectl delete deployment ha-app --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
