#!/bin/bash
set -e

echo "Setting up host-network task..."

kubectl create namespace host-net-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
