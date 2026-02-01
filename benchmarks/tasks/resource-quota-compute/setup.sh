#!/bin/bash
set -e

echo "Setting up resource-quota-compute task..."

kubectl create namespace quota-compute-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
