#!/bin/bash
set -e

echo "Setting up custom-dns-config task..."

kubectl create namespace dns-config-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
