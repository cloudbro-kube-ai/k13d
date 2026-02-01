#!/bin/bash
echo "Cleaning up limit-range-pod task..."
kubectl delete namespace limitrange-pod-test --ignore-not-found=true
echo "Cleanup complete."
