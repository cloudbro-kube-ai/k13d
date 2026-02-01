#!/bin/bash
# Setup script for priority-class task

set -e

echo "Setting up priority-class task..."

# Create namespace if not exists
kubectl create namespace schedule-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete pod critical-pod --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete priorityclass high-priority --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
