#!/bin/bash
echo "Cleaning up copy-files-to-pod task..."
kubectl delete namespace cp-test --ignore-not-found=true
echo "Cleanup complete."
