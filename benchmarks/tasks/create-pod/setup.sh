#!/bin/bash
# Setup script for create-pod task
# Ensures clean state before test

set -e

echo "Setting up create-pod task..."

# Delete any existing pod with the same name
kubectl delete pod web-server --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

# Wait for pod to be deleted
sleep 2

echo "Setup complete."
