#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create deployment
kubectl create deployment web-app --image=nginx:alpine -n benchmark --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: deployment 'web-app' created in namespace 'benchmark'"
