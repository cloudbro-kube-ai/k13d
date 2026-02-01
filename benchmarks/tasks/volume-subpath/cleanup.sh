#!/bin/bash
# Cleanup script for volume-subpath task

echo "Cleaning up volume-subpath task..."

kubectl delete pod subpath-pod --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete configmap app-files --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace volume-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
