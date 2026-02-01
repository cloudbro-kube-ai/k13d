#!/bin/bash
# Setup script for cronjob-concurrency task

set -e

echo "Setting up cronjob-concurrency task..."

# Create namespace if not exists
kubectl create namespace job-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing cronjob
kubectl delete cronjob report-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
