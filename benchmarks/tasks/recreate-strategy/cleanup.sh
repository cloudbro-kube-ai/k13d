#!/bin/bash
# Cleanup script for recreate-strategy task

echo "Cleaning up recreate-strategy task..."

kubectl delete deployment stateful-app --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace deploy-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
