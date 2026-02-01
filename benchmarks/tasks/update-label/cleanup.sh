#!/bin/bash
set -e

# Delete the pod if it exists
kubectl delete pod app-pod -n benchmark --ignore-not-found=true

echo "Cleanup complete"
