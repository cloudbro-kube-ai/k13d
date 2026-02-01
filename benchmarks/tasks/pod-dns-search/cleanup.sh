#!/bin/bash
echo "Cleaning up pod-dns-search task..."
kubectl delete namespace dns-search-test --ignore-not-found=true
echo "Cleanup complete."
