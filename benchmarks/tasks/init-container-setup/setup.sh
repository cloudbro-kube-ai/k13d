#!/bin/bash
set -e

echo "Setting up init-container-setup task..."

kubectl create namespace init-setup-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
