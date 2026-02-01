#!/bin/bash
echo "Cleaning up init-container-shared-volume task..."
kubectl delete namespace init-volume-test --ignore-not-found=true
echo "Cleanup complete."
