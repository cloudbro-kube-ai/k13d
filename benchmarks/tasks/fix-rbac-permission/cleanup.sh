#!/bin/bash
# Cleanup script for fix-rbac-permission task

set -e

echo "Cleaning up fix-rbac-permission task..."

kubectl delete pod api-client --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete serviceaccount api-client-sa --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete role --all --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete rolebinding --all --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
