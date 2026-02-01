#!/bin/bash
echo "Cleaning up init-container-setup task..."
kubectl delete namespace init-setup-test --ignore-not-found=true
echo "Cleanup complete."
