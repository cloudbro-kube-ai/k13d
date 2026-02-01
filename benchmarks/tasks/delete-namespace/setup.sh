#!/bin/bash
set -e

# Create the namespace to be deleted
kubectl create namespace deprecated-ns --dry-run=client -o yaml | kubectl apply -f -

# Create another namespace that should NOT be deleted
kubectl create namespace keep-ns --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete: namespaces 'deprecated-ns' and 'keep-ns' created"
