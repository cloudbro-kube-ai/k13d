#!/bin/bash
set -e

echo "Setting up dns-policy task..."

kubectl create namespace dns-policy-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
