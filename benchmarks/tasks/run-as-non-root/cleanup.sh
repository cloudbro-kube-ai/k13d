#!/bin/bash
echo "Cleaning up run-as-non-root task..."
kubectl delete namespace nonroot-test --ignore-not-found=true
echo "Cleanup complete."
