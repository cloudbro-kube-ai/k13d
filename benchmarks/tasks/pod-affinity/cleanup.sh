#!/bin/bash
# Cleanup script for pod-affinity task

echo "Cleaning up pod-affinity task..."

kubectl delete pod cache-pod --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment web-app --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace schedule-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
