#!/bin/bash
# Cleanup script for create-pvc task

set -e

echo "Cleaning up create-pvc task..."

kubectl delete pvc data-pvc --namespace="${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

echo "Cleanup complete."
