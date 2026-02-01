#!/bin/bash
set -e

echo "Setting up init-container-wait task..."

kubectl create namespace init-wait-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
