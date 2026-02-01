#!/bin/bash
set -e

# Delete the Deployment if it exists
kubectl delete deployment nginx-deployment -n benchmark --ignore-not-found=true

echo "Cleanup complete"
