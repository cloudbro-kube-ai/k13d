#!/bin/bash
set -e

# Create the development namespace
kubectl create namespace development --dry-run=client -o yaml | kubectl apply -f -

# Store the current namespace for cleanup
CURRENT_NS=$(kubectl config view --minify -o jsonpath='{..namespace}')
echo "$CURRENT_NS" > /tmp/original-namespace.txt

echo "Setup complete: namespace 'development' created"
