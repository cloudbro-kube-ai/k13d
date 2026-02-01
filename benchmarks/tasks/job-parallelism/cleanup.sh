#!/bin/bash
# Cleanup script for job-parallelism task

echo "Cleaning up job-parallelism task..."

kubectl delete job parallel-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace job-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
