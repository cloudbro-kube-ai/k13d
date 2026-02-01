#!/bin/bash
# Setup script for mount-projected-volume task

set -e

echo "Setting up mount-projected-volume task..."

# Create namespace if not exists
kubectl create namespace volume-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources with the same names
kubectl delete pod projected-pod --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete configmap app-config --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete secret app-secret --namespace=volume-test --ignore-not-found=true 2>/dev/null || true

# Wait for resources to be deleted
sleep 2

echo "Setup complete."
