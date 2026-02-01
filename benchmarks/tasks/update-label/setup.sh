#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create pod with version=v1 label
kubectl run app-pod --image=nginx:alpine -n benchmark --labels="app=myapp,version=v1" --restart=Never --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: pod 'app-pod' created with 'version=v1' label in namespace 'benchmark'"
