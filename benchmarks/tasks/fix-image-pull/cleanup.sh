#!/bin/bash
# Cleanup script for fix-image-pull task

echo "Cleaning up fix-image-pull task..."

kubectl delete deployment image-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
