#!/bin/bash
# Cleanup script for maxsurge-config task

echo "Cleaning up maxsurge-config task..."

kubectl delete deployment web-surge --namespace=deploy-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace deploy-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
