#!/bin/bash
set -e

# Delete the namespace if it exists
kubectl delete namespace production --ignore-not-found=true

echo "Cleanup complete"
