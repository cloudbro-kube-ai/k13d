#!/bin/bash
# Cleanup script for create-secret-env task

set -e

echo "Cleaning up create-secret-env task..."

kubectl delete pod app-pod --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete secret app-secrets --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
