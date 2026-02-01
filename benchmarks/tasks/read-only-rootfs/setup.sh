#!/bin/bash
set -e

echo "Setting up read-only-rootfs task..."

kubectl create namespace readonly-fs-test --dry-run=client -o yaml | kubectl apply -f -

echo "Setup complete. Namespace created."
