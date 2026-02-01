#!/bin/bash
echo "Cleaning up set-resource-limits task..."
kubectl delete namespace resource-lim-test --ignore-not-found=true
echo "Cleanup complete."
