#!/bin/bash
echo "Cleaning up container-logs-tail task..."
kubectl delete namespace logs-tail-test --ignore-not-found=true
echo "Cleanup complete."
