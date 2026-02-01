#!/bin/bash
set -e

# Delete the ConfigMap if it exists
kubectl delete configmap settings -n benchmark --ignore-not-found=true

echo "Cleanup complete"
