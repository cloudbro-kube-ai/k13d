#!/bin/bash
echo "Cleaning up sidecar-logging task..."
kubectl delete namespace sidecar-log-test --ignore-not-found=true
echo "Cleanup complete."
