#!/bin/bash
set -e

# Delete Services if they exist
kubectl delete service old-service keep-service -n benchmark --ignore-not-found=true

echo "Cleanup complete"
