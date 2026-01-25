#!/bin/bash
echo "Cleaning up create-network-policy task..."
kubectl delete namespace secure-app --ignore-not-found=true
echo "Cleanup complete."
