#!/bin/bash
# Cleanup script for priority-class task

echo "Cleaning up priority-class task..."

kubectl delete pod critical-pod --namespace=schedule-test --ignore-not-found=true 2>/dev/null || true
kubectl delete priorityclass high-priority --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace schedule-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
