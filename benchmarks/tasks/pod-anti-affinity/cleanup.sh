#!/bin/bash
# Cleanup script for pod-anti-affinity task

echo "Cleaning up pod-anti-affinity task..."

kubectl delete deployment ha-app --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace schedule-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
