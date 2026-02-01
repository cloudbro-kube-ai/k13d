#!/bin/bash
# Setup script for pod-affinity task

set -e

echo "Setting up pod-affinity task..."

# Create namespace if not exists
kubectl create namespace schedule-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete pod cache-pod --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment web-app --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
