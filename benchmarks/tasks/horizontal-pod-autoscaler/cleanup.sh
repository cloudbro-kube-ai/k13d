#!/bin/bash
echo "Cleaning up horizontal-pod-autoscaler task..."
kubectl delete namespace autoscale-ns --ignore-not-found=true
echo "Cleanup complete."
