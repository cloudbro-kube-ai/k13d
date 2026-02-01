#!/bin/bash
echo "Cleaning up init-container-dependency task..."
kubectl delete namespace init-dep-test --ignore-not-found=true
echo "Cleanup complete."
