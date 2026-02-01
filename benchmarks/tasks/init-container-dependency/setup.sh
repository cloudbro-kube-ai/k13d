#!/bin/bash
set -e

echo "Setting up init-container-dependency task..."

kubectl create namespace init-dep-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
