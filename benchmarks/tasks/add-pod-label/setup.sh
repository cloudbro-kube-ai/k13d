#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create target pod
kubectl run target-pod --image=nginx:alpine -n benchmark --restart=Never --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: pod 'target-pod' created in namespace 'benchmark'"
