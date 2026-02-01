#!/bin/bash
echo "Cleaning up previous-container-logs task..."
kubectl delete namespace prev-logs-test --ignore-not-found=true
echo "Cleanup complete."
