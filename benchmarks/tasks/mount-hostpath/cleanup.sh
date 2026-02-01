#!/bin/bash
# Cleanup script for mount-hostpath task

echo "Cleaning up mount-hostpath task..."

kubectl delete pod hostpath-pod --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace volume-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
