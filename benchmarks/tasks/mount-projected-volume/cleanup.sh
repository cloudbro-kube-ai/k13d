#!/bin/bash
# Cleanup script for mount-projected-volume task

echo "Cleaning up mount-projected-volume task..."

kubectl delete pod projected-pod --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete configmap app-config --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete secret app-secret --namespace=volume-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace volume-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
