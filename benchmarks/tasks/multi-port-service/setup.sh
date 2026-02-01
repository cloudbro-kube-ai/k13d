#!/bin/bash
# Setup script for multi-port-service task

set -e

echo "Setting up multi-port-service task..."

# Create namespace if not exists
kubectl create namespace service-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete service web-service --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment web-app --namespace=service-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
