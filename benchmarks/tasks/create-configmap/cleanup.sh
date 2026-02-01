#!/bin/bash
set -e

# Delete the ConfigMap if it exists
kubectl delete configmap app-config -n benchmark --ignore-not-found=true

echo "Cleanup complete"
