#!/bin/bash
echo "Cleaning up host-aliases task..."
kubectl delete namespace host-alias-test --ignore-not-found=true
echo "Cleanup complete."
