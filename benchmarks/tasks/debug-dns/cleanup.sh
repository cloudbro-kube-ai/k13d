#!/bin/bash
# Cleanup script for debug-dns task

set -e

echo "Cleaning up debug-dns task..."

kubectl delete pod dns-test --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete networkpolicy deny-dns --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
