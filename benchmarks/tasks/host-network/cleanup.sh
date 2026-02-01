#!/bin/bash
echo "Cleaning up host-network task..."
kubectl delete namespace host-net-test --ignore-not-found=true
echo "Cleanup complete."
