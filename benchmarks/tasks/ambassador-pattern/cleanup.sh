#!/bin/bash
echo "Cleaning up ambassador-pattern task..."
kubectl delete namespace ambassador-test --ignore-not-found=true
echo "Cleanup complete."
