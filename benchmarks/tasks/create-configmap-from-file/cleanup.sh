#!/bin/bash
set -e

# Delete the ConfigMap if it exists
kubectl delete configmap nginx-config -n benchmark --ignore-not-found=true

# Delete the temp file
rm -f /tmp/nginx.conf

echo "Cleanup complete"
