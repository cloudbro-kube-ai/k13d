#!/bin/bash
echo "Cleaning up init-container-wait task..."
kubectl delete namespace init-wait-test --ignore-not-found=true
echo "Cleanup complete."
