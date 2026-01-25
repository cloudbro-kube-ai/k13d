#!/bin/bash
echo "Cleaning up fix-service-routing task..."
kubectl delete namespace web --ignore-not-found=true
echo "Cleanup complete."
