#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the ConfigMap with initial values
kubectl create configmap settings \
    --from-literal=MAX_CONNECTIONS=100 \
    --from-literal=TIMEOUT=30 \
    --from-literal=RETRY_COUNT=3 \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: ConfigMap 'settings' created with MAX_CONNECTIONS=100"
