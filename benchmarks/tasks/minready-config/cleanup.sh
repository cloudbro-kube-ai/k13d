#!/bin/bash
# Cleanup script for minready-config task

echo "Cleaning up minready-config task..."

kubectl delete deployment stable-app --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace deploy-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
