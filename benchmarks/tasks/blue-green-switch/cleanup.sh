#!/bin/bash
# Cleanup script for blue-green-switch task

echo "Cleaning up blue-green-switch task..."

kubectl delete deployment app-blue app-green --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete service app-service --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace deploy-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
