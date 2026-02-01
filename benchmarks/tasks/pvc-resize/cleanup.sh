#!/bin/bash
# Cleanup script for pvc-resize task

echo "Cleaning up pvc-resize task..."

kubectl delete pvc data-pvc --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace volume-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
