#!/bin/bash
set -e

echo "Setting up sidecar-proxy task..."

kubectl create namespace sidecar-proxy-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
