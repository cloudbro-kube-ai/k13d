#!/bin/bash
set -e

# Delete the pod if it exists
kubectl delete pod readonly-pod -n benchmark --ignore-not-found=true

echo "Cleanup complete"
