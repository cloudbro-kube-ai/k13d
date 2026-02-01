#!/bin/bash
set -e

# Create some test namespaces
kubectl create namespace test-ns-alpha --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace test-ns-beta --dry-run=client -o yaml | kubectl apply -f -

# Remove any existing output file
rm -f /tmp/namespaces.txt

echo "Setup complete: test namespaces created"
