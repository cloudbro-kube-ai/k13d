#!/bin/bash
# Cleanup script for external-name-service task

echo "Cleaning up external-name-service task..."

kubectl delete service external-db --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace service-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
