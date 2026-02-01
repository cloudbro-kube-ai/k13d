#!/bin/bash
echo "Cleaning up resource-quota-compute task..."
kubectl delete namespace quota-compute-test --ignore-not-found=true
echo "Cleanup complete."
