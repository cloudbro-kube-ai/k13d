#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the ConfigMap to be deleted
kubectl create configmap old-config \
    --from-literal=KEY1=value1 \
    --from-literal=KEY2=value2 \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create another ConfigMap that should NOT be deleted
kubectl create configmap keep-config \
    --from-literal=IMPORTANT=yes \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: ConfigMaps 'old-config' and 'keep-config' created"
