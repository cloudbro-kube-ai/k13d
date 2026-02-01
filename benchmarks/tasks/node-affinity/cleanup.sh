#!/bin/bash
# Cleanup script for node-affinity task

echo "Cleaning up node-affinity task..."

kubectl delete pod affinity-pod --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace schedule-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
