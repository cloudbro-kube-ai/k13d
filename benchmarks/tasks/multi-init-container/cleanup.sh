#!/bin/bash
echo "Cleaning up multi-init-container task..."
kubectl delete namespace multi-init-test --ignore-not-found=true
echo "Cleanup complete."
