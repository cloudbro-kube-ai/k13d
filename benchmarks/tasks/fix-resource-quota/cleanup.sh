#!/bin/bash
# Cleanup script for fix-resource-quota task

set -e

echo "Cleaning up fix-resource-quota task..."

kubectl delete deployment web-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete resourcequota compute-quota --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
