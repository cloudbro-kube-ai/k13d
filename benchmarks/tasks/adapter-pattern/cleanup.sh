#!/bin/bash
echo "Cleaning up adapter-pattern task..."
kubectl delete namespace adapter-test --ignore-not-found=true
echo "Cleanup complete."
