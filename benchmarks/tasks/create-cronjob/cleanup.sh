#!/bin/bash
# Cleanup script for create-cronjob task

set -e

echo "Cleaning up create-cronjob task..."

kubectl delete cronjob backup-job --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
