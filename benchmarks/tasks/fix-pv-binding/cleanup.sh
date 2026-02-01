#!/bin/bash
# Cleanup script for fix-pv-binding task

set -e

echo "Cleaning up fix-pv-binding task..."

kubectl delete pvc app-data --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete pv task-pv-volume --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
