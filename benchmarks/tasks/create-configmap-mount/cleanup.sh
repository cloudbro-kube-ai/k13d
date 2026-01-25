#!/bin/bash
echo "Cleaning up create-configmap-mount task..."
kubectl delete namespace config-test --ignore-not-found=true
echo "Cleanup complete."
