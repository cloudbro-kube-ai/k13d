#!/bin/bash
set -e

# Delete ConfigMaps
kubectl delete configmap config-alpha config-beta config-gamma -n benchmark --ignore-not-found=true

# Remove output file
rm -f /tmp/configmaps.txt

echo "Cleanup complete"
