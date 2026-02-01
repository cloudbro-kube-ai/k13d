#!/bin/bash
# Cleanup script for cronjob-concurrency task

echo "Cleaning up cronjob-concurrency task..."

kubectl delete cronjob report-job --namespace=job-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace job-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
