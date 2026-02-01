#!/bin/bash
echo "Cleaning up drop-capabilities task..."
kubectl delete namespace drop-cap-test --ignore-not-found=true
echo "Cleanup complete."
