#!/bin/bash
# Cleanup script for exec-probe task

echo "Cleaning up exec-probe task..."

kubectl delete pod exec-probe-pod --namespace=probe-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace probe-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
