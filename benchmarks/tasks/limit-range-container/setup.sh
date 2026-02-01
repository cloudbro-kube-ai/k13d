#!/bin/bash
set -e

echo "Setting up limit-range-container task..."

kubectl create namespace limitrange-container-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
