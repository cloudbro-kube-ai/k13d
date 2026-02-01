#!/bin/bash
# Cleanup script for cronjob-suspend task

echo "Cleaning up cronjob-suspend task..."

kubectl delete cronjob cleanup-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace job-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
