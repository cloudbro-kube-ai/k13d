#!/bin/bash
set -e

# Delete the Service and Pod if they exist
kubectl delete service web-nodeport -n benchmark --ignore-not-found=true
kubectl delete pod web-pod -n benchmark --ignore-not-found=true

echo "Cleanup complete"
