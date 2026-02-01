#!/bin/bash
set -e

# Delete the Service and Pod if they exist
kubectl delete service backend-svc -n benchmark --ignore-not-found=true
kubectl delete pod backend-pod -n benchmark --ignore-not-found=true

echo "Cleanup complete"
