#!/bin/bash
set -e

# Delete ConfigMaps if they exist
kubectl delete configmap old-config keep-config -n benchmark --ignore-not-found=true

echo "Cleanup complete"
