#!/bin/bash
# Cleanup script for pause-rollout task

echo "Cleaning up pause-rollout task..."

kubectl delete deployment rolling-app --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace deploy-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
