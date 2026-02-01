#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the Secret to be deleted
kubectl create secret generic deprecated-secret \
    --from-literal=OLD_KEY=old-value \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create another Secret that should NOT be deleted
kubectl create secret generic active-secret \
    --from-literal=ACTIVE_KEY=active-value \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: Secrets 'deprecated-secret' and 'active-secret' created"
