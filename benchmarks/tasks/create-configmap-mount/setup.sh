#!/bin/bash
set -e

echo "Setting up create-configmap-mount task..."

kubectl create namespace config-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created, no ConfigMap or Pod exists yet."
