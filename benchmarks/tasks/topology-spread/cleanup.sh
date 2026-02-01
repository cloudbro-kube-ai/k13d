#!/bin/bash
# Cleanup script for topology-spread task

echo "Cleaning up topology-spread task..."

kubectl delete deployment spread-app --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace schedule-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
