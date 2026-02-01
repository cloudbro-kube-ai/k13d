#!/bin/bash
set -e

echo "Setting up set-resource-requests task..."

kubectl create namespace resource-req-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
