#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the Deployment to be deleted
kubectl create deployment old-deployment --image=nginx:alpine -n benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create another Deployment that should NOT be deleted
kubectl create deployment active-deployment --image=nginx:alpine -n benchmark --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: Deployments 'old-deployment' and 'active-deployment' created"
