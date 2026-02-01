#!/bin/bash
# Cleanup script for create-headless-service task

echo "Cleaning up create-headless-service task..."

kubectl delete service db-headless --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete deployment database --namespace=service-test --ignore-not-found=true 2>/dev/null || true
kubectl delete namespace service-test --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
