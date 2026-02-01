#!/bin/bash
# Cleanup script for readiness-probe-tcp task

echo "Cleaning up readiness-probe-tcp task..."

kubectl delete pod readiness-tcp --namespace=probe-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace probe-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
