#!/bin/bash
echo "Cleaning up events-filter task..."
kubectl delete namespace events-test --ignore-not-found=true
echo "Cleanup complete."
