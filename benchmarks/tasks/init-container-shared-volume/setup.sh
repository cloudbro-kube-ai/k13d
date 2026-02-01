#!/bin/bash
set -e

echo "Setting up init-container-shared-volume task..."

kubectl create namespace init-volume-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
