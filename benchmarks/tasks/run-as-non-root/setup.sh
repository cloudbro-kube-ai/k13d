#!/bin/bash
set -e

echo "Setting up run-as-non-root task..."

kubectl create namespace nonroot-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
