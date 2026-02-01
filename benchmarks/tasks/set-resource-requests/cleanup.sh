#!/bin/bash
echo "Cleaning up set-resource-requests task..."
kubectl delete namespace resource-req-test --ignore-not-found=true
echo "Cleanup complete."
