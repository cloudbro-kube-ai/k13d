#!/bin/bash
set -e

# Delete the deployment if it exists
kubectl delete deployment web-app -n benchmark --ignore-not-found=true

echo "Cleanup complete"
