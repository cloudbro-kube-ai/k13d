#!/bin/bash
echo "Cleaning up sidecar-proxy task..."
kubectl delete namespace sidecar-proxy-test --ignore-not-found=true
echo "Cleanup complete."
