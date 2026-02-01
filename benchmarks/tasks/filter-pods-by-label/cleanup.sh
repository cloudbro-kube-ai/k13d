#!/bin/bash
set -e

# Delete all pods
kubectl delete pod frontend-1 frontend-2 backend-1 backend-2 -n benchmark --ignore-not-found=true

# Clean up output file
rm -f /tmp/frontend-pods.txt

echo "Cleanup complete"
