#!/bin/bash
echo "Cleaning up custom-dns-config task..."
kubectl delete namespace dns-config-test --ignore-not-found=true
echo "Cleanup complete."
