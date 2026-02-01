#!/bin/bash
# Cleanup script for create-ingress task

set -e

echo "Cleaning up create-ingress task..."

kubectl delete ingress app-ingress --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete service api-svc --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
kubectl delete service web-svc --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
