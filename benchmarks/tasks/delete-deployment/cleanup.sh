#!/bin/bash
set -e

# Delete Deployments if they exist
kubectl delete deployment old-deployment active-deployment -n benchmark --ignore-not-found=true

echo "Cleanup complete"
