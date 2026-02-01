#!/bin/bash
set -e

echo "Setting up set-resource-limits task..."

kubectl create namespace resource-lim-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
