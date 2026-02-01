#!/bin/bash
# Cleanup script for service-topology task

echo "Cleaning up service-topology task..."

kubectl delete service local-svc --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment local-app --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace service-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
