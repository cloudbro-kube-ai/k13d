#!/bin/bash
set -e

echo "Setting up host-aliases task..."

kubectl create namespace host-alias-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
