#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create pods with different labels
kubectl run frontend-1 --image=nginx:alpine -n benchmark --labels="tier=frontend,app=web" --restart=Never --dry-run=client -o yaml | kubectl apply -f -
kubectl run frontend-2 --image=nginx:alpine -n benchmark --labels="tier=frontend,app=api" --restart=Never --dry-run=client -o yaml | kubectl apply -f -
kubectl run backend-1 --image=redis:alpine -n benchmark --labels="tier=backend,app=cache" --restart=Never --dry-run=client -o yaml | kubectl apply -f -
kubectl run backend-2 --image=busybox:1.35 -n benchmark --labels="tier=backend,app=worker" --restart=Never --command -- sleep 3600 --dry-run=client -o yaml | kubectl apply -f -

# Clean up any existing output file
rm -f /tmp/frontend-pods.txt

echo "Setup complete: pods created with different tier labels in namespace 'benchmark'"
