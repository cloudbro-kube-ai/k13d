#!/bin/bash
echo "Cleaning up exec-into-pod task..."
kubectl delete namespace exec-test --ignore-not-found=true
echo "Cleanup complete."
