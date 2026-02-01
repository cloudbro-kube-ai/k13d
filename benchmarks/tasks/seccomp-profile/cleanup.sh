#!/bin/bash
echo "Cleaning up seccomp-profile task..."
kubectl delete namespace seccomp-test --ignore-not-found=true
echo "Cleanup complete."
