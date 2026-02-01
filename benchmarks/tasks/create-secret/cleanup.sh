#!/bin/bash
set -e

# Delete the Secret if it exists
kubectl delete secret db-credentials -n benchmark --ignore-not-found=true

echo "Cleanup complete"
