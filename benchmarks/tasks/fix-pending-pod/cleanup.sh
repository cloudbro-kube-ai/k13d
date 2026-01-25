#!/bin/bash
echo "Cleaning up fix-pending-pod task..."
kubectl delete namespace homepage-ns --ignore-not-found=true
echo "Cleanup complete."
