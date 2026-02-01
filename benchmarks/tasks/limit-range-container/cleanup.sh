#!/bin/bash
echo "Cleaning up limit-range-container task..."
kubectl delete namespace limitrange-container-test --ignore-not-found=true
echo "Cleanup complete."
