#!/bin/bash
# Cleanup script for create-pod task

echo "Cleaning up create-pod task..."

kubectl delete pod web-server --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
