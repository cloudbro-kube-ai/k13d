#!/bin/bash
set -e

# Delete namespaces if they exist
kubectl delete namespace deprecated-ns keep-ns --ignore-not-found=true

echo "Cleanup complete"
