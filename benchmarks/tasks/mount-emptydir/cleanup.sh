#!/bin/bash
# Cleanup script for mount-emptydir task

echo "Cleaning up mount-emptydir task..."

kubectl delete pod shared-data --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace volume-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
