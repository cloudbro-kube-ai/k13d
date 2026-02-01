#!/bin/bash
set -e

echo "Setting up limit-range-pod task..."

kubectl create namespace limitrange-pod-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
