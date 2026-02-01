#!/bin/bash
# Cleanup script for probe-failure-threshold task

echo "Cleaning up probe-failure-threshold task..."

kubectl delete deployment resilient-app --namespace=probe-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace probe-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
