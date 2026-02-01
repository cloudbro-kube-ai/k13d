#!/bin/bash
echo "Cleaning up sidecar-sync task..."
kubectl delete namespace sidecar-sync-test --ignore-not-found=true
echo "Cleanup complete."
