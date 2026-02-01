#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create pod with the deprecated label
kubectl run legacy-pod --image=nginx:alpine -n benchmark --labels="app=legacy,deprecated=true,version=v1" --restart=Never --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: pod 'legacy-pod' created with 'deprecated' label in namespace 'benchmark'"
