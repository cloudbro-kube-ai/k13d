#!/bin/bash
# Setup script for create-headless-service task

set -e

echo "Setting up create-headless-service task..."

# Create namespace if not exists
kubectl create namespace service-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing resources
kubectl delete service db-headless --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment database --namespace=service-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
