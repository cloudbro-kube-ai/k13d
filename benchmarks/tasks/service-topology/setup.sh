#!/bin/bash
# Setup script for service-topology task

set -e

echo "Setting up service-topology task..."

# Create namespace if not exists
kubectl create namespace service-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete service local-svc --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment local-app --namespace=service-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
