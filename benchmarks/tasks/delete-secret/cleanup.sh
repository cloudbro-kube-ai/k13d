#!/bin/bash
set -e

# Delete Secrets if they exist
kubectl delete secret deprecated-secret active-secret -n benchmark --ignore-not-found=true

echo "Cleanup complete"
