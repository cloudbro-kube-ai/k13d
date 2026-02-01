#!/bin/bash
# Cleanup script for service-affinity task

echo "Cleaning up service-affinity task..."

kubectl delete service sticky-svc --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment sticky-app --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace service-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
