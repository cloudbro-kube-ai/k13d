#!/bin/bash
# Cleanup script for create-rbac-role task

set -e

echo "Cleaning up create-rbac-role task..."

kubectl delete serviceaccount app-reader --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete role pod-reader --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete rolebinding app-reader-binding --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete pod test-pod --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
