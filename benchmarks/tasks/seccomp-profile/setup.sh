#!/bin/bash
set -e

echo "Setting up seccomp-profile task..."

kubectl create namespace seccomp-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
