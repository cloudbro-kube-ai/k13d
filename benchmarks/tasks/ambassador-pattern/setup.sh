#!/bin/bash
set -e

echo "Setting up ambassador-pattern task..."

kubectl create namespace ambassador-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
