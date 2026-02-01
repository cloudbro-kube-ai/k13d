#!/bin/bash
set -e

echo "Setting up drop-capabilities task..."

kubectl create namespace drop-cap-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
