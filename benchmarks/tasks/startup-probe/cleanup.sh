#!/bin/bash
# Cleanup script for startup-probe task

echo "Cleaning up startup-probe task..."

kubectl delete pod startup-app --namespace=probe-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace probe-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
