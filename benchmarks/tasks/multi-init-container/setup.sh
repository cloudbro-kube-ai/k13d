#!/bin/bash
set -e

echo "Setting up multi-init-container task..."

kubectl create namespace multi-init-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
