#!/bin/bash
echo "Cleaning up dns-policy task..."
kubectl delete namespace dns-policy-test --ignore-not-found=true
echo "Cleanup complete."
