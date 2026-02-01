#!/bin/bash
# Cleanup script for job-backoff-limit task

echo "Cleaning up job-backoff-limit task..."

kubectl delete job retry-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace job-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
