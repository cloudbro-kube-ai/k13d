#!/bin/bash
# Cleanup script for multi-port-service task

echo "Cleaning up multi-port-service task..."

kubectl delete service web-service --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment web-app --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace service-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
