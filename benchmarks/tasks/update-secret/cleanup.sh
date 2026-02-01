#!/bin/bash
set -e

# Delete the Secret if it exists
kubectl delete secret api-secret -n benchmark --ignore-not-found=true

echo "Cleanup complete"
