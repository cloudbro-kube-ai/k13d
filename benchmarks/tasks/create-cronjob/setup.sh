#!/bin/bash
# Setup script for create-cronjob task

set -e

echo "Setting up create-cronjob task..."

# Delete any existing cronjob with the same name
kubectl delete cronjob backup-job --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

# Wait for cleanup
sleep 2

echo "Setup complete."
