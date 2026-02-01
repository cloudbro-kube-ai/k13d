#!/bin/bash
set -e

echo "Setting up sidecar-sync task..."

kubectl create namespace sidecar-sync-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
