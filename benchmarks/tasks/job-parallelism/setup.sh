#!/bin/bash
# Setup script for job-parallelism task

set -e

echo "Setting up job-parallelism task..."

# Create namespace if not exists
kubectl create namespace job-test --dry-run=client -o yaml | kubectl apply -f -

# Delete any existing job
kubectl delete job parallel-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true

sleep 2

echo "Setup complete."
