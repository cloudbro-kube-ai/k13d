#!/bin/bash
set -e

# Delete the ResourceQuota if it exists
kubectl delete resourcequota compute-quota -n benchmark --ignore-not-found=true

echo "Cleanup complete"
