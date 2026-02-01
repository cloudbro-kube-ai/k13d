#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the Secret with initial values
kubectl create secret generic api-secret \
    --from-literal=API_KEY=old-api-key-2023 \
    --from-literal=API_SECRET=keep-this-value \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: Secret 'api-secret' created with initial API_KEY"
