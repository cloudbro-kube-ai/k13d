#!/bin/bash
# Cleanup script for fix-crashloop task

echo "Cleaning up fix-crashloop task..."

kubectl delete deployment broken-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
