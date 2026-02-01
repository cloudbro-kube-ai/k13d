#!/bin/bash
set -e

echo "Setting up privileged-pod task..."

kubectl create namespace privileged-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
