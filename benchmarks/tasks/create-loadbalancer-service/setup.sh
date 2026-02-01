#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create a pod that the service will select
kubectl run api-pod --image=nginx:alpine -n benchmark --labels="app=api" --restart=Never --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: namespace 'benchmark' and pod 'api-pod' are ready"
