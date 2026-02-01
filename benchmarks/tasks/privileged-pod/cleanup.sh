#!/bin/bash
echo "Cleaning up privileged-pod task..."
kubectl delete namespace privileged-test --ignore-not-found=true
echo "Cleanup complete."
