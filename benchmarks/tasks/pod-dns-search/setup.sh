#!/bin/bash
set -e

echo "Setting up pod-dns-search task..."

kubectl create namespace dns-search-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
