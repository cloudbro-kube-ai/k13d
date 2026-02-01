#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create multiple ConfigMaps
kubectl create configmap config-alpha --from-literal=key=alpha -n benchmark --dry-run=client -o yaml | kubectl apply -f -
kubectl create configmap config-beta --from-literal=key=beta -n benchmark --dry-run=client -o yaml | kubectl apply -f -
kubectl create configmap config-gamma --from-literal=key=gamma -n benchmark --dry-run=client -o yaml | kubectl apply -f -

# Remove any existing output file
rm -f /tmp/configmaps.txt

echo "Setup complete: ConfigMaps created in namespace 'benchmark'"
