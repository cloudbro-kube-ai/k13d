#!/bin/bash
# Cleanup script for liveness-probe-http task

echo "Cleaning up liveness-probe-http task..."

kubectl delete pod liveness-http --namespace=probe-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace probe-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
