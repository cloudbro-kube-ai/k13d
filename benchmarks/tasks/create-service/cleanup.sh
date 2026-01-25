#!/bin/bash
# Cleanup script for create-service task

echo "Cleaning up create-service task..."

kubectl delete deployment backend-app --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete service backend-service --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
