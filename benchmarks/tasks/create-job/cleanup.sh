#!/bin/bash
# Cleanup script for create-job task

echo "Cleaning up create-job task..."

kubectl delete job data-processor --namespace=job-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace job-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
