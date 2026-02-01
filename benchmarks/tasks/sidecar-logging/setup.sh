#!/bin/bash
set -e

echo "Setting up sidecar-logging task..."

kubectl create namespace sidecar-log-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
