#!/bin/bash
set -e

# Create namespace for the benchmark
kubectl create namespace benchmark --dry-run=client -o yaml | kubectl apply -f -

# Create the Secret with an encoded password
kubectl create secret generic encoded-secret \
    --from-literal=username=myuser \
    --from-literal=password=SuperSecret123! \
    -n benchmark --dry-run=client -o yaml | kubectl apply -f -

# Remove any existing output file
rm -f /tmp/decoded-password.txt

echo "Setup complete: Secret 'encoded-secret' created in namespace 'benchmark'"
