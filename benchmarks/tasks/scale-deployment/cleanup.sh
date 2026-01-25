#!/bin/bash
# Cleanup script for scale-deployment task

echo "Cleaning up scale-deployment task..."

kubectl delete deployment web-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
