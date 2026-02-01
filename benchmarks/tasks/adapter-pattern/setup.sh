#!/bin/bash
set -e

echo "Setting up adapter-pattern task..."

kubectl create namespace adapter-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
